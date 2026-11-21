package ingest

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"
)

// MalformedLogFile is the name of the malformed-row report inside the output directory.
const MalformedLogFile = "malformed_rows.csv"

// MalformedLog writes rejected rows as CSV: line_number,reason,raw_line.
type MalformedLog struct {
	file *os.File
	csv  *csv.Writer
}

// CreateMalformedLog creates (or truncates) path and writes the header row.
func CreateMalformedLog(path string) (*MalformedLog, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	l := &MalformedLog{file: f, csv: csv.NewWriter(f)}
	if err := l.csv.Write([]string{"line_number", "reason", "raw_line"}); err != nil {
		f.Close()
		return nil, err
	}
	return l, nil
}

// Write appends one rejected row. It matches the onMalformed callback of Parse.
func (l *MalformedLog) Write(m Malformed) error {
	return l.csv.Write([]string{strconv.FormatInt(m.Line, 10), m.Reason, string(m.Raw)})
}

// Close flushes buffered rows and closes the file.
func (l *MalformedLog) Close() error {
	l.csv.Flush()
	return errors.Join(l.csv.Error(), l.file.Close())
}
