package ingest

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

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

	var gotIDs []string
	var gotMalformed []Malformed
	stats, err := Parse(strings.NewReader(input),
		func(r Record) { gotIDs = append(gotIDs, string(r.CampaignID)) },
		func(m Malformed) error {
			m.Raw = bytes.Clone(m.Raw)
			gotMalformed = append(gotMalformed, m)
			return nil
		})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if stats != (Stats{Rows: 2, Malformed: 3}) {
		t.Errorf("stats = %+v, want {Rows:2 Malformed:3}", stats)
	}
	if strings.Join(gotIDs, ",") != "CMP001,CMP001" {
		t.Errorf("records = %v, want [CMP001 CMP001]", gotIDs)
	}
	wantMalformed := []struct {
		line   int64
		reason string
		raw    string
	}{
		{3, "empty line", ""},
		{4, "invalid impressions", "CMP002,2025-01-01,x,10,1.50,1"},
		{5, "line longer than 65536 bytes", longLine[:rawPrefixLen]},
	}
	if len(gotMalformed) != len(wantMalformed) {
		t.Fatalf("got %d malformed rows, want %d: %+v", len(gotMalformed), len(wantMalformed), gotMalformed)
	}
	for i, w := range wantMalformed {
		g := gotMalformed[i]
		if g.Line != w.line || g.Reason != w.reason || string(g.Raw) != w.raw {
			t.Errorf("malformed[%d] = {%d %q %.40q}, want {%d %q %.40q}", i, g.Line, g.Reason, g.Raw, w.line, w.reason, w.raw)
		}
	}
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.input), func(Record) {}, func(Malformed) error { return nil })
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestParseStopsOnMalformedHandlerError(t *testing.T) {
	sentinel := errors.New("disk full")
	input := Header + "\nbad\nCMP001,2025-01-01,100,10,1.50,1\n"
	stats, err := Parse(strings.NewReader(input), func(Record) {}, func(Malformed) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
	if stats.Rows != 0 {
		t.Errorf("rows parsed after handler error: %d", stats.Rows)
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

func BenchmarkParse(b *testing.B) {
	var buf bytes.Buffer
	buf.WriteString(Header + "\r\n")
	for i := 0; i < 100_000; i++ {
		buf.WriteString("CMP020,2025-05-03,24465,764,1394.62,42\r\n")
	}
	data := buf.Bytes()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Parse(bytes.NewReader(data), func(Record) {}, func(Malformed) error { return nil }); err != nil {
			b.Fatal(err)
		}
	}
}

func recordsEqual(a, b Record) bool {
	return bytes.Equal(a.CampaignID, b.CampaignID) && a.Impressions == b.Impressions &&
		a.Clicks == b.Clicks && a.SpendCents == b.SpendCents && a.Conversions == b.Conversions
}
