package ingest

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
)

const (
	defaultChunkSize = 1 << 20   // bytes handed to a worker at a time
	maxLineBytes     = 64 * 1024 // longer lines are malformed
	maxHeaderBytes   = 4096      // a longer first line cannot be the header
)

// ErrEmptyInput is returned when the input has no header line.
var ErrEmptyInput = errors.New("input is empty")

// errStopped tells the reader that the collector gave up early.
var errStopped = errors.New("parse stopped")

// Accumulator receives the valid rows parsed by one worker.
type Accumulator interface {
	Add(Record)
}

// Stats counts the data rows seen by Parse.
type Stats struct {
	Rows      int64 // valid rows added to an accumulator
	Malformed int64 // rows passed to onMalformed
}

// limits are the size knobs of the pipeline; tests shrink them to hit chunk edges.
type limits struct {
	chunkSize int // must exceed maxLine
	maxLine   int
}

// Parse validates the header, then parses the data rows on the given number of
// worker goroutines. Each worker folds its valid rows into its own accumulator
// from newAcc; the accumulators are returned for the caller to merge.
//
// onMalformed is called from the calling goroutine, once per rejected row, in
// input order. An error from it stops parsing and is returned.
//
// Memory is bounded: the input is read in fixed-size chunks from a pool of
// 2×workers buffers, whatever the file size.
func Parse[A Accumulator](r io.Reader, workers int, newAcc func() A, onMalformed func(Malformed) error) ([]A, Stats, error) {
	return parse(r, workers, limits{chunkSize: defaultChunkSize, maxLine: maxLineBytes}, newAcc, onMalformed)
}

// chunk is a run of whole lines for one worker. A chunk with tooLong set stands
// for one line longer than a buffer: the reader skipped it and kept its prefix.
type chunk struct {
	seq     int
	data    []byte
	buf     []byte // pooled buffer behind data, returned after parsing
	tooLong []byte
}

// chunkResult is what a worker reports for one chunk. Malformed line numbers
// are relative to the chunk (1 = its first line).
type chunkResult struct {
	seq       int
	lines     int64
	rows      int64
	malformed []Malformed
}

func parse[A Accumulator](r io.Reader, workers int, lim limits, newAcc func() A, onMalformed func(Malformed) error) ([]A, Stats, error) {
	workers = max(workers, 1)
	br := bufio.NewReaderSize(r, maxHeaderBytes)
	if err := readHeader(br); err != nil {
		return nil, Stats{}, err
	}

	free := make(chan []byte, 2*workers)
	for range 2 * workers {
		free <- make([]byte, lim.chunkSize)
	}
	chunks := make(chan chunk, workers)
	results := make(chan chunkResult, workers)
	done := make(chan struct{})

	readErr := make(chan error, 1)
	go func() {
		defer close(chunks)
		readErr <- readChunks(br, lim, free, chunks, done)
	}()

	accs := make([]A, workers)
	var wg sync.WaitGroup
	for i := range accs {
		accs[i] = newAcc()
		wg.Add(1)
		go func(acc A) {
			defer wg.Done()
			for c := range chunks {
				res := parseChunk(c, acc, lim.maxLine)
				if c.buf != nil {
					free <- c.buf
				}
				select {
				case results <- res:
				case <-done:
				}
			}
		}(accs[i])
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	stats, err := collect(results, onMalformed)
	if err != nil {
		close(done)
		for range results { // let the workers finish
		}
		<-readErr
		return nil, stats, err
	}
	if err := <-readErr; err != nil {
		return nil, stats, fmt.Errorf("read input: %w", err)
	}
	return accs, stats, nil
}

// collect puts chunk results back in input order, so malformed rows get their
// absolute line numbers and reach onMalformed in file order.
func collect(results <-chan chunkResult, onMalformed func(Malformed) error) (Stats, error) {
	var stats Stats
	pending := make(map[int]chunkResult)
	next, lineBase := 0, int64(1) // the header is line 1
	for res := range results {
		pending[res.seq] = res
		for r, ok := pending[next]; ok; r, ok = pending[next] {
			delete(pending, next)
			next++
			for _, m := range r.malformed {
				m.Line += lineBase
				if err := onMalformed(m); err != nil {
					return stats, err
				}
				stats.Malformed++
			}
			lineBase += r.lines
			stats.Rows += r.rows
		}
	}
	return stats, nil
}

// readChunks cuts r into chunks that end on a line boundary. The partial last
// line of each buffer is carried into the next one.
func readChunks(r io.Reader, lim limits, free chan []byte, out chan<- chunk, done <-chan struct{}) error {
	take := func() ([]byte, error) {
		select {
		case b := <-free:
			return b, nil
		case <-done:
			return nil, errStopped
		}
	}
	seq := 0
	send := func(c chunk) error {
		c.seq = seq
		seq++
		select {
		case out <- c:
			return nil
		case <-done:
			return errStopped
		}
	}

	buf, err := take()
	if err != nil {
		return err
	}
	n := 0 // bytes carried over at the start of buf
	for {
		m, err := io.ReadFull(r, buf[n:])
		n += m
		eof := errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
		if err != nil && !eof {
			return err
		}
		if eof {
			if n == 0 {
				free <- buf
				return nil
			}
			return send(chunk{data: buf[:n], buf: buf})
		}

		last := bytes.LastIndexByte(buf, '\n')
		if last < 0 { // a single line fills the whole buffer
			prefix := rawPrefix(buf)
			var atEOF bool
			if n, atEOF, err = skipLine(r, buf); err != nil {
				return err
			}
			if err := send(chunk{tooLong: prefix}); err != nil || atEOF {
				free <- buf
				return err
			}
			continue
		}

		next, err := take()
		if err != nil {
			return err
		}
		n = copy(next, buf[last+1:])
		if err := send(chunk{data: buf[:last+1], buf: buf}); err != nil {
			return err
		}
		buf = next
	}
}

// skipLine discards the rest of an over-long line, using buf as scratch. It
// returns how many bytes after the line's terminator it left at the start of buf.
func skipLine(r io.Reader, buf []byte) (carried int, atEOF bool, err error) {
	for {
		m, err := io.ReadFull(r, buf)
		if i := bytes.IndexByte(buf[:m], '\n'); i >= 0 {
			return copy(buf, buf[i+1:m]), false, nil
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return 0, true, nil
		}
		if err != nil {
			return 0, false, err
		}
	}
}

// parseChunk parses the lines of one chunk into acc.
func parseChunk(c chunk, acc Accumulator, maxLine int) chunkResult {
	res := chunkResult{seq: c.seq}
	tooLongReason := fmt.Sprintf("line longer than %d bytes", maxLine)
	if c.tooLong != nil {
		res.lines = 1
		res.malformed = []Malformed{{Line: 1, Reason: tooLongReason, Raw: c.tooLong}}
		return res
	}
	for data := c.data; len(data) > 0; {
		var line []byte
		line, data, _ = bytes.Cut(data, []byte("\n"))
		line = trimCR(line)
		res.lines++

		if len(line) > maxLine {
			res.malformed = append(res.malformed, Malformed{Line: res.lines, Reason: tooLongReason, Raw: rawPrefix(line)})
			continue
		}
		rec, reason := parseRecord(line)
		if reason != "" {
			res.malformed = append(res.malformed, Malformed{Line: res.lines, Reason: reason, Raw: bytes.Clone(line)})
			continue
		}
		res.rows++
		acc.Add(rec)
	}
	return res
}

func readHeader(br *bufio.Reader) error {
	line, err := br.ReadSlice('\n')
	switch {
	case errors.Is(err, io.EOF) && len(line) == 0:
		return ErrEmptyInput
	case errors.Is(err, bufio.ErrBufferFull):
		return fmt.Errorf("unexpected header: first line is longer than %d bytes", br.Size())
	case err != nil && !errors.Is(err, io.EOF):
		return fmt.Errorf("read header: %w", err)
	}
	line = trimCR(bytes.TrimSuffix(line, []byte("\n")))
	line = bytes.TrimPrefix(line, []byte("\xef\xbb\xbf"))
	if string(line) != Header {
		return fmt.Errorf("unexpected header %q, want %q", line, Header)
	}
	return nil
}

func trimCR(line []byte) []byte {
	return bytes.TrimSuffix(line, []byte("\r"))
}

// rawPrefix copies the start of an over-long line for the malformed-row log.
func rawPrefix(line []byte) []byte {
	return bytes.Clone(line[:min(rawPrefixLen, len(line))])
}
