package api

import (
	"encoding/json"
	"log"

	"github.com/thrasher-redhat/internal-tools/pkg/bugzilla"
)

type BugResolver struct {
	bug bugzilla.Bug
}

func (r *BugResolver) DateStamp() string {
	return r.bug.DateStamp.Format(dateFormat)
}

func (r *BugResolver) ID() int32 {
	return int32(r.bug.ID)
}

func (r *BugResolver) Component() string {
	return string(r.bug.Component)
}

func (r *BugResolver) Status() string {
	return r.bug.Status
}

func (r *BugResolver) Summary() string {
	return r.bug.Summary
}

func (r *BugResolver) TargetRelease() string {
	return string(r.bug.TargetRelease)
}

func (r *BugResolver) AssignedTo() string {
	return r.bug.AssignedTo
}

func (r *BugResolver) Keywords() []string {
	return r.bug.Keywords
}

// PmScore is an integer represention of priority
func (r *BugResolver) PmScore() int32 {
	return int32(r.bug.PmScore)
}

// external is for parsing the json 'externals' object
type external struct {
	ExtBzID int `json:"ext_bz_id"`
}

// CustomerCase returns if there are any customer cases associated with the bug
func (r *BugResolver) CustomerCase() bool {
	// Unmarshal the json with the only field we need
	var e []external
	err := json.Unmarshal(r.bug.Externals, &e)
	if err != nil {
		log.Printf("Unable to unmarshal externals json: %v\n", err)
		return false
	}

	for _, ex := range e {
		if ex.ExtBzID == bugzilla.ExternalID {
			return true
		}
	}
	return false
}

// Age is the number of days since the bug id was first seen
func (r *BugResolver) Age() int32 {
	return int32(r.bug.Age)
}
