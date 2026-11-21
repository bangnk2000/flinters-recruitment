package report

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"adagg/internal/agg"
)

func campaign(id string, impressions, clicks, spendCents, conversions int64) agg.Campaign {
	return agg.Campaign{ID: id, Totals: agg.Totals{
		Impressions: impressions, Clicks: clicks, SpendCents: spendCents, Conversions: conversions,
	}}
}

func ids(cs []agg.Campaign) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

func TestTopByCTROrdersAndTruncates(t *testing.T) {
	var cs []agg.Campaign
	for i := 1; i <= 12; i++ { // CMP01 has CTR 0.01 … CMP12 has CTR 0.12
		cs = append(cs, campaign(fmt.Sprintf("CMP%02d", i), 1000, int64(10*i), 100, 1))
	}
	input := slices.Clone(cs)

	got := ids(TopByCTR(cs, TopN))
	want := []string{"CMP12", "CMP11", "CMP10", "CMP09", "CMP08", "CMP07", "CMP06", "CMP05", "CMP04", "CMP03"}
	if !slices.Equal(got, want) {
		t.Errorf("top = %v, want %v", got, want)
	}
	if !slices.Equal(ids(cs), ids(input)) {
		t.Error("TopByCTR reordered its input")
	}
}

func TestTopByCTRTieBreaksByID(t *testing.T) {
	cs := []agg.Campaign{
		campaign("CMP_C", 2000, 100, 0, 0), // 0.05
		campaign("CMP_A", 1000, 50, 0, 0),  // 0.05
		campaign("CMP_D", 1000, 90, 0, 0),  // 0.09
		campaign("CMP_B", 4000, 200, 0, 0), // 0.05
	}
	got := ids(TopByCTR(cs, TopN))
	want := []string{"CMP_D", "CMP_A", "CMP_B", "CMP_C"}
	if !slices.Equal(got, want) {
		t.Errorf("top = %v, want %v", got, want)
	}
}

func TestTopByCTRFewerThanN(t *testing.T) {
	if got := TopByCTR(nil, TopN); len(got) != 0 {
		t.Errorf("top of nothing = %v, want empty", got)
	}
}

func TestTopByCPAOrdersAndTruncates(t *testing.T) {
	var cs []agg.Campaign
	for i := 12; i >= 1; i-- { // CMPxx has CPA = xx dollars
		cs = append(cs, campaign(fmt.Sprintf("CMP%02d", i), 1000, 10, int64(i)*100, 1))
	}
	input := slices.Clone(cs)

	got := ids(TopByCPA(cs, TopN))
	want := []string{"CMP01", "CMP02", "CMP03", "CMP04", "CMP05", "CMP06", "CMP07", "CMP08", "CMP09", "CMP10"}
	if !slices.Equal(got, want) {
		t.Errorf("top = %v, want %v", got, want)
	}
	if !slices.Equal(ids(cs), ids(input)) {
		t.Error("TopByCPA reordered its input")
	}
}

func TestTopByCPAExcludesZeroConversionsAndTieBreaksByID(t *testing.T) {
	cs := []agg.Campaign{
		campaign("CMP_ZERO", 1000, 10, 1, 0), // excluded despite the tiny spend
		campaign("CMP_C", 1000, 10, 2000, 2), // 10.00
		campaign("CMP_A", 1000, 10, 1000, 1), // 10.00
		campaign("CMP_D", 1000, 10, 500, 1),  // 5.00
		campaign("CMP_B", 1000, 10, 3000, 3), // 10.00
	}
	got := ids(TopByCPA(cs, TopN))
	want := []string{"CMP_D", "CMP_A", "CMP_B", "CMP_C"}
	if !slices.Equal(got, want) {
		t.Errorf("top = %v, want %v", got, want)
	}
}

func TestTopByCPAAllZeroConversions(t *testing.T) {
	cs := []agg.Campaign{campaign("CMP1", 10, 1, 100, 0)}
	if got := TopByCPA(cs, TopN); len(got) != 0 {
		t.Errorf("top = %v, want empty", ids(got))
	}
}

func TestWriteCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), TopCTRFile)
	err := WriteCSV(path, []agg.Campaign{
		campaign("CMP042", 125000, 6250, 1250050, 625),
		campaign("CMP004", 9000, 90, 2000, 0),
		campaign("CMP005", 3, 1, 7, 3),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "campaign_id,total_impressions,total_clicks,total_spend,total_conversions,CTR,CPA\n" +
		"CMP042,125000,6250,12500.50,625,0.0500,20.00\n" +
		"CMP004,9000,90,20.00,0,0.0100,\n" +
		"CMP005,3,1,0.07,3,0.3333,0.02\n"
	if string(got) != want {
		t.Errorf("csv =\n%s\nwant\n%s", got, want)
	}
}

func TestWriteCSVFailsOnMissingDir(t *testing.T) {
	if err := WriteCSV(filepath.Join(t.TempDir(), "missing", TopCTRFile), nil); err == nil {
		t.Error("want error for a path in a missing directory")
	}
}

func TestFormatCents(t *testing.T) {
	for cents, want := range map[int64]string{0: "0.00", 5: "0.05", 50: "0.50", 4550: "45.50", 39582525210: "395825252.10"} {
		if got := formatCents(cents); got != want {
			t.Errorf("formatCents(%d) = %q, want %q", cents, got, want)
		}
	}
}
