package ingest

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// collector records campaign IDs in the order its worker saw them.
type collector struct{ ids []string }

func (c *collector) Add(r Record) { c.ids = append(c.ids, string(r.CampaignID)) }

// summer totals records per campaign, like the real aggregator.
type summer struct{ totals map[string][4]int64 }

func newSummer() *summer { return &summer{totals: make(map[string][4]int64)} }

func (s *summer) Add(r Record) {
	t := s.totals[string(r.CampaignID)]
	s.totals[string(r.CampaignID)] = [4]int64{t[0] + r.Impressions, t[1] + r.Clicks, t[2] + r.SpendCents, t[3] + r.Conversions}
}

type parseOutcome struct {
	stats     Stats
	malformed []Malformed
	totals    map[string][4]int64
}

func runParse(t *testing.T, input string, workers int, lim limits) parseOutcome {
	t.Helper()
	var out parseOutcome
	accs, stats, err := parse(strings.NewReader(input), workers, lim, newSummer, func(m Malformed) error {
		out.malformed = append(out.malformed, m)
		return nil
	})
	if err != nil {
		t.Fatalf("parse(workers=%d, %+v): %v", workers, lim, err)
	}
	out.stats = stats
	out.totals = make(map[string][4]int64)
	for _, acc := range accs {
		for id, v := range acc.totals {
			t := out.totals[id]
			out.totals[id] = [4]int64{t[0] + v[0], t[1] + v[1], t[2] + v[2], t[3] + v[3]}
		}
	}
	return out
}

func TestParseRecord(t *testing.T) {
	tests := []struct {
		line       string
		want       Record
		wantReason string
	}{
		{line: "CMP001,2025-01-01,12000,300,45.50,12", want: Record{CampaignID: []byte("CMP001"), Impressions: 12000, Clicks: 300, SpendCents: 4550, Conversions: 12}},
		{line: "CMP002,2024-02-29,0,0,45.5,0", want: Record{CampaignID: []byte("CMP002"), SpendCents: 4550}},
		{line: "CMP003,2025-12-31,1,1,45,1", want: Record{CampaignID: []byte("CMP003"), Impressions: 1, Clicks: 1, SpendCents: 4500, Conversions: 1}},
		{line: "", wantReason: "empty line"},
		{line: "CMP001,2025-01-01,12000,300,45.50", wantReason: "expected 6 fields, got 5"},
		{line: "CMP001,2025-01-01,12000,300,45.50,12,x", wantReason: "expected 6 fields, got 7"},
		{line: `"CMP001,X",2025-01-01,1,1,1,1`, wantReason: "expected 6 fields, got 7"},
		{line: ",2025-01-01,12000,300,45.50,12", wantReason: "empty campaign_id"},
		{line: "CMP001,2025-1-01,12000,300,45.50,12", wantReason: "invalid date"},
		{line: "CMP001,2025-13-01,12000,300,45.50,12", wantReason: "invalid date"},
		{line: "CMP001,2025-02-29,12000,300,45.50,12", wantReason: "invalid date"},
		{line: "CMP001,01/01/2025,12000,300,45.50,12", wantReason: "invalid date"},
		{line: "CMP001,2025-01-01,-5,300,45.50,12", wantReason: "invalid impressions"},
		{line: "CMP001,2025-01-01,12000,abc,45.50,12", wantReason: "invalid clicks"},
		{line: "CMP001,2025-01-01,12000,300,,12", wantReason: "invalid spend"},
		{line: "CMP001,2025-01-01,12000,300,45.505,12", wantReason: "invalid spend"},
		{line: "CMP001,2025-01-01,12000,300,45.,12", wantReason: "invalid spend"},
		{line: "CMP001,2025-01-01,12000,300,.5,12", wantReason: "invalid spend"},
		{line: "CMP001,2025-01-01,12000,300,-1.00,12", wantReason: "invalid spend"},
		{line: "CMP001,2025-01-01,12000,300,45.50,1.5", wantReason: "invalid conversions"},
		{line: "CMP001,2025-01-01,1234567890123456789,300,45.50,12", wantReason: "invalid impressions"},
	}
	for _, tc := range tests {
		t.Run(tc.line, func(t *testing.T) {
			got, reason := parseRecord([]byte(tc.line))
			if reason != tc.wantReason {
				t.Fatalf("reason = %q, want %q", reason, tc.wantReason)
			}
			if reason == "" && !recordsEqual(got, tc.want) {
				t.Errorf("record = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseStream(t *testing.T) {
	longLine := "CMP009,2025-01-01," + strings.Repeat("9", maxLineBytes)
	input := "\xef\xbb\xbf" + Header + "\r\n" +
		"CMP001,2025-01-01,100,10,1.50,1\r\n" + // line 2
		"\r\n" + // line 3: empty
		"CMP002,2025-01-01,x,10,1.50,1\r\n" + // line 4: invalid impressions
		longLine + "\n" + // line 5: too long
		"CMP001,2025-01-02,200,20,2.25,2" // line 6: no trailing newline

	var gotMalformed []Malformed
	accs, stats, err := Parse(strings.NewReader(input), 1,
		func() *collector { return &collector{} },
		func(m Malformed) error {
			gotMalformed = append(gotMalformed, m)
			return nil
		})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if stats != (Stats{Rows: 2, Malformed: 3}) {
		t.Errorf("stats = %+v, want {Rows:2 Malformed:3}", stats)
	}
	if len(accs) != 1 || strings.Join(accs[0].ids, ",") != "CMP001,CMP001" {
		t.Errorf("records = %v, want one accumulator with [CMP001 CMP001]", accs)
	}
	want := []Malformed{
		{Line: 3, Reason: "empty line", Raw: []byte{}},
		{Line: 4, Reason: "invalid impressions", Raw: []byte("CMP002,2025-01-01,x,10,1.50,1")},
		{Line: 5, Reason: "line longer than 65536 bytes", Raw: []byte(longLine[:rawPrefixLen])},
	}
	if !reflect.DeepEqual(gotMalformed, want) {
		t.Errorf("malformed =\n%.200q\nwant\n%.200q", gotMalformed, want)
	}
}

// TestParseWorkersAgree is the safety net for the parallel pipeline: any number
// of workers and any chunk size must give exactly the single-worker result, with
// rows, line terminators and over-long lines falling on every chunk boundary.
func TestParseWorkersAgree(t *testing.T) {
	input := generatedInput(3000)
	const maxLine = 48
	want := runParse(t, input, 1, limits{chunkSize: 1 << 20, maxLine: maxLine})
	if want.stats.Malformed == 0 || want.stats.Rows == 0 || len(want.totals) == 0 {
		t.Fatalf("generated input is not mixed enough: %+v", want.stats)
	}

	for _, workers := range []int{1, 2, 8} {
		for _, chunkSize := range []int{rawPrefixLen, 300, 4096} {
			t.Run(fmt.Sprintf("workers=%d/chunk=%d", workers, chunkSize), func(t *testing.T) {
				got := runParse(t, input, workers, limits{chunkSize: chunkSize, maxLine: maxLine})
				if got.stats != want.stats {
					t.Errorf("stats = %+v, want %+v", got.stats, want.stats)
				}
				if !reflect.DeepEqual(got.malformed, want.malformed) {
					t.Errorf("malformed rows differ from the single-worker run (%d vs %d rows)", len(got.malformed), len(want.malformed))
				}
				if !reflect.DeepEqual(got.totals, want.totals) {
					t.Errorf("totals = %v, want %v", got.totals, want.totals)
				}
			})
		}
	}
}

// generatedInput builds a deterministic file mixing valid rows with every kind
// of malformed row, LF and CRLF terminators, and over-long lines; the last line
// is over-long and has no terminator.
func generatedInput(lines int) string {
	rng := rand.New(rand.NewPCG(1, 2))
	var b strings.Builder
	b.WriteString(Header + "\n")
	for range lines {
		switch rng.IntN(20) {
		case 0:
			// empty line
		case 1:
			b.WriteString("CMP01,2025-01-01,1,1")
		case 2:
			b.WriteString(strings.Repeat("x", 60)) // between maxLine and the chunk size
		case 3:
			b.WriteString(strings.Repeat("y", 700)) // longer than the small chunks
		default:
			fmt.Fprintf(&b, "CMP%02d,2025-01-%02d,%d,%d,%d.%02d,%d", rng.IntN(7), 1+rng.IntN(28),
				rng.IntN(100000), rng.IntN(1000), rng.IntN(1000), rng.IntN(100), rng.IntN(100))
		}
		if rng.IntN(2) == 0 {
			b.WriteString("\r\n")
		} else {
			b.WriteString("\n")
		}
	}
	b.WriteString(strings.Repeat("z", 700))
	return b.String()
}

func TestParseHeaderErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"empty input", "", ErrEmptyInput.Error()},
		{"wrong header", "id,date,impressions,clicks,spend,conversions\n", "unexpected header"},
		{"data without header", "CMP001,2025-01-01,100,10,1.50,1\n", "unexpected header"},
		{"huge first line", strings.Repeat("h", maxHeaderBytes+1) + "\n", "first line is longer than"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Parse(strings.NewReader(tc.input), 2, newSummer, func(Malformed) error { return nil })
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseHeaderOnly(t *testing.T) {
	got := runParse(t, Header+"\r\n", 4, limits{chunkSize: defaultChunkSize, maxLine: maxLineBytes})
	if got.stats != (Stats{}) || len(got.malformed) != 0 || len(got.totals) != 0 {
		t.Errorf("header-only input gave %+v", got)
	}
}

// TestParseStopsOnMalformedHandlerError also proves the pipeline shuts down
// (no goroutine stays blocked) when the collector gives up mid-stream.
func TestParseStopsOnMalformedHandlerError(t *testing.T) {
	sentinel := errors.New("disk full")
	input := Header + "\n" + strings.Repeat("bad\nCMP001,2025-01-01,100,10,1.50,1\n", 5000)
	calls := 0
	_, _, err := parse(strings.NewReader(input), 4, limits{chunkSize: 256, maxLine: 48}, newSummer,
		func(Malformed) error {
			if calls++; calls == 3 {
				return sentinel
			}
			return nil
		})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
	if calls != 3 {
		t.Errorf("onMalformed called %d times, want it to stop at 3", calls)
	}
}

func TestParseRecordDoesNotAllocate(t *testing.T) {
	line := []byte("CMP001,2025-01-01,12000,300,45.50,12")
	allocs := testing.AllocsPerRun(1000, func() {
		if _, reason := parseRecord(line); reason != "" {
			t.Fatal(reason)
		}
	})
	if allocs != 0 {
		t.Errorf("parseRecord allocates %.0f times per valid row, want 0", allocs)
	}
}

type discard struct{}

func (discard) Add(Record) {}

func BenchmarkParse(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteString(Header + "\r\n")
	for range 400_000 {
		buf.WriteString("CMP020,2025-05-03,24465,764,1394.62,42\r\n")
	}
	data := buf.Bytes()
	for _, workers := range []int{1, runtime.GOMAXPROCS(0)} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()
			for b.Loop() {
				if _, _, err := Parse(bytes.NewReader(data), workers, func() discard { return discard{} }, func(Malformed) error { return nil }); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func recordsEqual(a, b Record) bool {
	return bytes.Equal(a.CampaignID, b.CampaignID) && a.Impressions == b.Impressions &&
		a.Clicks == b.Clicks && a.SpendCents == b.SpendCents && a.Conversions == b.Conversions
}
