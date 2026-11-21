// Package report ranks aggregated campaigns and writes the result CSV files.
package report

import (
	"cmp"
	"slices"
	"strings"

	"adagg/internal/agg"
)

// TopN is the number of campaigns in each ranking.
const TopN = 10

// TopByCTR returns up to n campaigns with the highest CTR. Equal CTRs are
// ordered by campaign ID ascending so the output is deterministic.
func TopByCTR(campaigns []agg.Campaign, n int) []agg.Campaign {
	ranked := slices.Clone(campaigns)
	slices.SortFunc(ranked, func(a, b agg.Campaign) int {
		return cmp.Or(cmp.Compare(b.CTR(), a.CTR()), strings.Compare(a.ID, b.ID))
	})
	return ranked[:min(n, len(ranked))]
}

// TopByCPA returns up to n campaigns with the lowest CPA, excluding campaigns
// without conversions. Equal CPAs are ordered by campaign ID ascending.
func TopByCPA(campaigns []agg.Campaign, n int) []agg.Campaign {
	ranked := slices.DeleteFunc(slices.Clone(campaigns), func(c agg.Campaign) bool {
		return c.Conversions == 0
	})
	slices.SortFunc(ranked, func(a, b agg.Campaign) int {
		cpaA, _ := a.CPA()
		cpaB, _ := b.CPA()
		return cmp.Or(cmp.Compare(cpaA, cpaB), strings.Compare(a.ID, b.ID))
	})
	return ranked[:min(n, len(ranked))]
}
