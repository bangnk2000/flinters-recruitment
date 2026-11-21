// Package ingest streams ad-performance CSV rows and validates each one.
//
// The input is read line by line through a fixed-size buffer, so memory use does
// not grow with file size, and a valid row is parsed without heap allocation.
// Fields are split on commas without quote handling: the schema has no free-text
// columns, and a quoted row is reported as malformed rather than misread.
package ingest

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"
)

// Header is the exact first line the input must have.
const Header = "campaign_id,date,impressions,clicks,spend,conversions"

const (
	numFields    = 6
	maxLineBytes = 64 * 1024
	maxCountLen  = 18 // digits that always fit in an int64
	rawPrefixLen = 256
)

// ErrEmptyInput is returned when the input has no header line.
var ErrEmptyInput = errors.New("input is empty")

// Record is one valid data row. CampaignID aliases the read buffer and is only
// valid during the onRecord call; copy it to keep it.
type Record struct {
	CampaignID  []byte
	Impressions int64
	Clicks      int64
	SpendCents  int64
	Conversions int64
}

// Malformed is a rejected data row. Line is the 1-based physical line number
// (the header is line 1). Raw aliases the read buffer, like Record.CampaignID.
type Malformed struct {
	Line   int64
	Reason string
	Raw    []byte
}

// Stats counts the data rows seen by Parse.
type Stats struct {
	Rows      int64 // valid rows passed to onRecord
	Malformed int64 // rows passed to onMalformed
}

// Parse reads the header, then calls onRecord for every valid row and
// onMalformed for every invalid one. An error from onMalformed stops parsing.
func Parse(r io.Reader, onRecord func(Record), onMalformed func(Malformed) error) (Stats, error) {
	br := bufio.NewReaderSize(r, maxLineBytes)
	if err := readHeader(br); err != nil {
		return Stats{}, err
	}

	var stats Stats
	for lineNo := int64(2); ; lineNo++ {
		line, tooLong, err := readLine(br)
		if errors.Is(err, io.EOF) {
			return stats, nil
		}
		if err != nil {
			return stats, fmt.Errorf("read line %d: %w", lineNo, err)
		}

		var rec Record
		var reason string
		if tooLong {
			reason = fmt.Sprintf("line longer than %d bytes", maxLineBytes)
		} else {
			rec, reason = parseRecord(line)
		}
		if reason != "" {
			stats.Malformed++
			if err := onMalformed(Malformed{Line: lineNo, Reason: reason, Raw: line}); err != nil {
				return stats, err
			}
			continue
		}
		stats.Rows++
		onRecord(rec)
	}
}

func readHeader(br *bufio.Reader) error {
	line, _, err := readLine(br)
	if errors.Is(err, io.EOF) {
		return ErrEmptyInput
	}
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	line = bytes.TrimPrefix(line, []byte("\xef\xbb\xbf"))
	if string(line) != Header {
		return fmt.Errorf("unexpected header %q, want %q", line, Header)
	}
	return nil
}

// readLine returns the next line without its "\n" or "\r\n" terminator. A line
// longer than the buffer is consumed whole and returned as a copy of its first
// rawPrefixLen bytes with tooLong set. It returns io.EOF only when no bytes remain.
func readLine(br *bufio.Reader) (line []byte, tooLong bool, err error) {
	line, err = br.ReadSlice('\n')
	if errors.Is(err, bufio.ErrBufferFull) {
		prefix := bytes.Clone(line[:rawPrefixLen])
		for errors.Is(err, bufio.ErrBufferFull) {
			_, err = br.ReadSlice('\n')
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, true, err
		}
		return prefix, true, nil
	}
	if errors.Is(err, io.EOF) {
		if len(line) == 0 {
			return nil, false, io.EOF
		}
		err = nil // last line has no terminator
	}
	if err != nil {
		return nil, false, err
	}
	line = bytes.TrimSuffix(line, []byte("\n"))
	line = bytes.TrimSuffix(line, []byte("\r"))
	return line, false, nil
}

// parseRecord validates one data line. A non-empty reason means the row is malformed.
func parseRecord(line []byte) (Record, string) {
	if len(line) == 0 {
		return Record{}, "empty line"
	}

	var fields [numFields][]byte
	n := 0
	for rest, more := line, true; more; n++ {
		var field []byte
		field, rest, more = bytes.Cut(rest, []byte(","))
		if n < numFields {
			fields[n] = field
		}
	}
	if n != numFields {
		return Record{}, fmt.Sprintf("expected %d fields, got %d", numFields, n)
	}

	rec := Record{CampaignID: fields[0]}
	var ok bool
	switch {
	case len(fields[0]) == 0:
		return Record{}, "empty campaign_id"
	case !validDate(fields[1]):
		return Record{}, "invalid date"
	}
	if rec.Impressions, ok = parseCount(fields[2]); !ok {
		return Record{}, "invalid impressions"
	}
	if rec.Clicks, ok = parseCount(fields[3]); !ok {
		return Record{}, "invalid clicks"
	}
	if rec.SpendCents, ok = parseCents(fields[4]); !ok {
		return Record{}, "invalid spend"
	}
	if rec.Conversions, ok = parseCount(fields[5]); !ok {
		return Record{}, "invalid conversions"
	}
	return rec, ""
}

// parseCount parses a non-negative decimal integer of at most maxCountLen digits.
func parseCount(b []byte) (int64, bool) {
	if len(b) == 0 || len(b) > maxCountLen {
		return 0, false
	}
	var n int64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	return n, true
}

// parseCents parses a non-negative amount with at most two decimals ("45", "45.5", "45.50") into cents.
func parseCents(b []byte) (int64, bool) {
	whole, frac, hasDot := bytes.Cut(b, []byte("."))
	if hasDot && (len(frac) == 0 || len(frac) > 2) {
		return 0, false
	}
	units, ok := parseCount(whole)
	if !ok || len(whole) > maxCountLen-2 {
		return 0, false
	}
	var cents int64
	if hasDot {
		if cents, ok = parseCount(frac); !ok {
			return 0, false
		}
		if len(frac) == 1 {
			cents *= 10
		}
	}
	return units*100 + cents, true
}

// validDate reports whether b is a real calendar date in YYYY-MM-DD form.
func validDate(b []byte) bool {
	if len(b) != 10 || b[4] != '-' || b[7] != '-' {
		return false
	}
	year, okY := parseCount(b[0:4])
	month, okM := parseCount(b[5:7])
	day, okD := parseCount(b[8:10])
	if !okY || !okM || !okD || month < 1 || month > 12 || day < 1 {
		return false
	}
	daysInMonth := time.Date(int(year), time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return int(day) <= daysInMonth
}
