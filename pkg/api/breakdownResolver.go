package api

import (
	"github.com/thrasher-redhat/internal-tools/pkg/db"
)

// Breakdown of a given bug query
// Contains Total bugs, new bugs, and closed bugs

type BreakdownResolver struct {
	breakdown db.Breakdown
}

// Total is the number of bugs in the given query
func (r *BreakdownResolver) Total() int32 {
	return int32(r.breakdown.Total)
}

// New is the number of bugs in the query that were not there the previous date (that we have data for)
func (r *BreakdownResolver) New() int32 {
	return int32(r.breakdown.New)
}

// Closed is the number of bugs in the query for the previous date (that we have data for) that aren't there in the current query
func (r *BreakdownResolver) Closed() int32 {
	return int32(r.breakdown.Closed)
}
