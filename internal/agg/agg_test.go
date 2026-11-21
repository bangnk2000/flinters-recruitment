package agg

import (
	"math"
	"os"
	"testing"

	"adagg/internal/ingest"
)

// TestAggregateSample parses testdata/sample.csv end to end; the expected totals
// are summed by hand from its seven valid rows.
func TestAggregateSample(t *testing.T) {
	f, err := os.Open("../../testdata/sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	a := New()
	if _, err := ingest.Parse(f, a.Add, func(ingest.Malformed) error { return nil }); err != nil {
		t.Fatal(err)
	}

	want := []Campaign{
		{"CMP001", Totals{Impressions: 26000, Clicks: 640, SpendCents: 9370, Conversions: 27}},
		{"CMP002", Totals{Impressions: 16500, Clicks: 270, SpendCents: 5900, Conversions: 9}},
		{"CMP003", Totals{Impressions: 11000, Clicks: 150, SpendCents: 3375, Conversions: 9}},
		{"CMP004", Totals{Impressions: 9000, Clicks: 90, SpendCents: 2000, Conversions: 0}},
	}
	got := a.Campaigns()
	if len(got) != len(want) {
		t.Fatalf("got %d campaigns, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("campaign[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestAddCopiesCampaignID(t *testing.T) {
	buf := []byte("CMP001")
	a := New()
	a.Add(ingest.Record{CampaignID: buf, Impressions: 1})
	copy(buf, "XXXXXX") // the parser reuses its buffer for the next line
	a.Add(ingest.Record{CampaignID: []byte("CMP001"), Impressions: 2})

	got := a.Campaigns()
	if len(got) != 1 || got[0].ID != "CMP001" || got[0].Impressions != 3 {
		t.Errorf("campaigns = %+v, want one CMP001 with 3 impressions", got)
	}
}

func TestAddDoesNotAllocateForKnownCampaign(t *testing.T) {
	a := New()
	rec := ingest.Record{CampaignID: []byte("CMP001"), Impressions: 1}
	a.Add(rec)
	if allocs := testing.AllocsPerRun(1000, func() { a.Add(rec) }); allocs != 0 {
		t.Errorf("Add allocates %.0f times for a known campaign, want 0", allocs)
	}
}

func TestDerivedMetrics(t *testing.T) {
	tests := []struct {
		name    string
		totals  Totals
		wantCTR float64
		wantCPA float64
		wantOK  bool
	}{
		{"brief example", Totals{Impressions: 125000, Clicks: 6250, SpendCents: 1250050, Conversions: 625}, 0.05, 20.0008, true},
		{"zero conversions", Totals{Impressions: 9000, Clicks: 90, SpendCents: 2000}, 0.01, 0, false},
		{"zero impressions", Totals{SpendCents: 500, Conversions: 2}, 0, 2.5, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := Campaign{ID: "X", Totals: tc.totals}
			if got := c.CTR(); !almostEqual(got, tc.wantCTR) {
				t.Errorf("CTR = %v, want %v", got, tc.wantCTR)
			}
			cpa, ok := c.CPA()
			if ok != tc.wantOK || !almostEqual(cpa, tc.wantCPA) {
				t.Errorf("CPA = %v, %v; want %v, %v", cpa, ok, tc.wantCPA, tc.wantOK)
			}
		})
	}
}

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
