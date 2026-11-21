package ingest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMalformedLogQuotesRawLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), MalformedLogFile)
	log, err := CreateMalformedLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.Write(Malformed{Line: 7, Reason: "invalid spend", Raw: []byte(`CMP1,2025-01-01,1,1,"x",1`)}); err != nil {
		t.Fatal(err)
	}
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "line_number,reason,raw_line\n" +
		`7,invalid spend,"CMP1,2025-01-01,1,1,""x"",1"` + "\n"
	if string(got) != want {
		t.Errorf("log =\n%s\nwant\n%s", got, want)
	}
}

func TestCreateMalformedLogFailsOnMissingDir(t *testing.T) {
	if _, err := CreateMalformedLog(filepath.Join(t.TempDir(), "missing", MalformedLogFile)); err == nil {
		t.Error("want error for a path in a missing directory")
	}
}
