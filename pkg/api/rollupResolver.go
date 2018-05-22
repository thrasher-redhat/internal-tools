package api

//import ()

// A single rollup

type RollupResolver struct {
	datestamp     string
	all           *BreakdownResolver
	blockers      *BreakdownResolver
	customerCases *BreakdownResolver
}

func (r *RollupResolver) Datestamp() string {
	return r.datestamp
}

func (r *RollupResolver) All() *BreakdownResolver {
	return r.all
}

func (r *RollupResolver) Blockers() *BreakdownResolver {
	return r.blockers
}

func (r *RollupResolver) CustomerCases() *BreakdownResolver {
	return r.customerCases
}
