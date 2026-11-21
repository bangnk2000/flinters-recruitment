package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "in.csv")
	if err := os.WriteFile(csvPath, []byte("campaign_id,date,impressions,clicks,spend,conversions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(dir, "empty.csv")
	if err := os.WriteFile(emptyPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	badHeaderPath := filepath.Join(dir, "bad-header.csv")
	if err := os.WriteFile(badHeaderPath, []byte("id,date\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join(dir, "missing.csv")
	newOutDir := filepath.Join(dir, "results", "nested")

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{"help", []string{"-h"}, ExitOK, "Usage: aggregator", ""},
		{"no args", nil, ExitUsage, "", "--input is required"},
		{"missing output", []string{"--input", csvPath}, ExitUsage, "", "--output is required"},
		{"unknown flag", []string{"--bogus"}, ExitUsage, "", "flag provided but not defined: -bogus"},
		{"positional arg", []string{"--input", csvPath, "--output", dir, "extra"}, ExitUsage, "", `unexpected argument "extra"`},
		{"input missing", []string{"--input", missingPath, "--output", dir}, ExitFault, "", "no such file or directory"},
		{"input is directory", []string{"--input", dir, "--output", dir}, ExitFault, "", "is a directory"},
		{"output is a file", []string{"--input", csvPath, "--output", csvPath}, ExitFault, "", "cannot create output directory"},
		{"empty input", []string{"--input", emptyPath, "--output", dir}, ExitFault, "", "input is empty"},
		{"wrong header", []string{"--input", badHeaderPath, "--output", dir}, ExitFault, "", "unexpected header"},
		{"success creates output dir", []string{"--input", csvPath, "--output", newOutDir}, ExitOK, "", "0 rows processed, 0 malformed rows skipped"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)

			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %q)", code, tc.wantCode, stderr.String())
			}
			assertContains(t, "stdout", stdout.String(), tc.wantStdout)
			assertContains(t, "stderr", stderr.String(), tc.wantStderr)
		})
	}

	if _, err := os.Stat(filepath.Join(newOutDir, "malformed_rows.csv")); err != nil {
		t.Errorf("malformed_rows.csv not created in %s: %v", newOutDir, err)
	}
}

// assertContains checks that got contains want; an empty want means got must be empty.
func assertContains(t *testing.T, stream, got, want string) {
	t.Helper()
	if want == "" {
		if got != "" {
			t.Errorf("%s = %q, want empty", stream, got)
		}
		return
	}
	if !strings.Contains(got, want) {
		t.Errorf("%s = %q, want it to contain %q", stream, got, want)
	}
}
