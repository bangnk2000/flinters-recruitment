// Package agg sums ad-performance records per campaign and derives CTR and CPA.
package agg

import (
	"slices"
	"strings"

	"adagg/internal/ingest"
)

// Totals are the summed metrics of one campaign. Spend is kept in integer
// cents so that summing millions of rows has no floating-point drift.
type Totals struct {
	Impressions int64
	Clicks      int64
	SpendCents  int64
	Conversions int64
}

// Campaign is one campaign's totals with its derived metrics.
type Campaign struct {
	ID string
	Totals
}

// CTR is clicks per impression, or 0 when the campaign has no impressions.
func (c Campaign) CTR() float64 {
	if c.Impressions == 0 {
		return 0
	}
	return float64(c.Clicks) / float64(c.Impressions)
}

// CPA is spend in USD per conversion. ok is false when there are no conversions.
func (c Campaign) CPA() (cpa float64, ok bool) {
	if c.Conversions == 0 {
		return 0, false
	}
	return float64(c.SpendCents) / 100 / float64(c.Conversions), true
}

// Aggregator accumulates records per campaign_id. Memory grows with the number
// of distinct campaigns, not with the number of rows.
type Aggregator struct {
	byID map[string]*Totals
}

// New returns an empty Aggregator.
func New() *Aggregator {
	return &Aggregator{byID: make(map[string]*Totals)}
}

// Add folds one record into its campaign's totals. It copies the campaign ID
// only the first time the campaign is seen, so repeat rows do not allocate.
func (a *Aggregator) Add(r ingest.Record) {
	t, ok := a.byID[string(r.CampaignID)]
	if !ok {
		t = &Totals{}
		a.byID[string(r.CampaignID)] = t
	}
	t.Impressions += r.Impressions
	t.Clicks += r.Clicks
	t.SpendCents += r.SpendCents
	t.Conversions += r.Conversions
}

// Campaigns returns every campaign's totals, ordered by campaign ID.
func (a *Aggregator) Campaigns() []Campaign {
	out := make([]Campaign, 0, len(a.byID))
	for id, t := range a.byID {
		out = append(out, Campaign{ID: id, Totals: *t})
	}
	slices.SortFunc(out, func(x, y Campaign) int { return strings.Compare(x.ID, y.ID) })
	return out
}
