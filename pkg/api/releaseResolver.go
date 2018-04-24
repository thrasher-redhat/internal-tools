package api

import (
	"github.com/thrasher-redhat/internal-tools/pkg/options"
)

type ReleaseResolver struct {
	release options.Release
	rollups []*RollupResolver
}

func (r *ReleaseResolver) Name() string {
	return r.release.Name
}

func (r *ReleaseResolver) Start() string {
	return r.release.Dates.Start
}

func (r *ReleaseResolver) GA() string {
	return r.release.Dates.Ga
}

func (r *ReleaseResolver) CodeFreeze() string {
	return r.release.Dates.CodeFreeze
}

func (r *ReleaseResolver) FeatureComplete() string {
	return r.release.Dates.FeatureComplete
}

func (r *ReleaseResolver) Rollups() []*RollupResolver {
	return r.rollups
}
