package report

import (
	"encoding/csv"
	"errors"
	"os"
	"strconv"

	"adagg/internal/agg"
)

// File names of the reports inside the output directory.
const (
	TopCTRFile = "top10_ctr.csv"
)

var header = []string{
	"campaign_id", "total_impressions", "total_clicks", "total_spend",
	"total_conversions", "CTR", "CPA",
}

// WriteCSV writes campaigns to path in the brief's column order. Spend is
// printed exactly from cents, CTR with 4 decimals, CPA with 2 decimals, and CPA
// is an empty cell for a campaign with no conversions.
func WriteCSV(path string, campaigns []agg.Campaign) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		f.Close()
		return err
	}
	for _, c := range campaigns {
		if err := w.Write(row(c)); err != nil {
			f.Close()
			return err
		}
	}
	w.Flush()
	return errors.Join(w.Error(), f.Close())
}

func row(c agg.Campaign) []string {
	cpa := ""
	if v, ok := c.CPA(); ok {
		cpa = strconv.FormatFloat(v, 'f', 2, 64)
	}
	return []string{
		c.ID,
		strconv.FormatInt(c.Impressions, 10),
		strconv.FormatInt(c.Clicks, 10),
		formatCents(c.SpendCents),
		strconv.FormatInt(c.Conversions, 10),
		strconv.FormatFloat(c.CTR(), 'f', 4, 64),
		cpa,
	}
}

// formatCents renders a non-negative cent amount as dollars with exactly two decimals.
func formatCents(cents int64) string {
	frac := strconv.FormatInt(cents%100, 10)
	if len(frac) == 1 {
		frac = "0" + frac
	}
	return strconv.FormatInt(cents/100, 10) + "." + frac
}
