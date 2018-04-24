package api

//import ()

// A Snapshot of a given day
// Will default to the latest snapshot
// Bug list will be used to calculate things on the frontend

type SnapshotResolver struct {
	datestamp string
	bugs      []*BugResolver
	rollup    *RollupResolver
}

func (r *SnapshotResolver) Datestamp() string {
	return r.datestamp
}

func (r *SnapshotResolver) Bugs() *[]*BugResolver {
	return &r.bugs
}

func (r *SnapshotResolver) Rollup() *RollupResolver {
	return r.rollup
}
