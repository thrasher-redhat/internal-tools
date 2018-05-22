package bugzilla

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/lib/pq"
)

// ExternalID is the specific ID for "Red Hat Customer Portal"
const ExternalID = 60

// SingleElemSlice deals with the special case where an [1]array is returned
type SingleElemSlice string

// MarshalJSON is a wrapper that converts a string to a single element slice
func (s SingleElemSlice) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{string(s)})
}

// UnmarshalJSON is a wrapper that pulls the string from a single element slice
func (s *SingleElemSlice) UnmarshalJSON(b []byte) error {
	var sl []string
	err := json.Unmarshal(b, &sl)
	if err != nil {
		return err
	}
	if len(sl) != 1 {
		return fmt.Errorf("expected len 1 for Single Element Slice, got %d", len(sl))
	}

	*s = SingleElemSlice(sl[0])
	return nil
}

// Score manages a string from json that needs to be an int
type Score int

// MarshalJSON is a wrapper that converts an int to a string
func (s Score) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// UnmarshalJSON is a wrapper that converts a string to an integer
func (s *Score) UnmarshalJSON(b []byte) error {
	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return err
	}
	*s = Score(i)
	return nil
}

// Bug maps to the desired fields of a bugzilla bug
type Bug struct {
	Id            int             `json:"id"`
	Component     SingleElemSlice `json:"component"`
	TargetRelease SingleElemSlice `json:"target_release"`
	AssignedTo    string          `json:"assigned_to"`
	Status        string          `json:"status"`
	Keywords      pq.StringArray  `json:"keywords"`
	PmScore       Score           `json:"cf_pm_score"`
	Summary       string          `json:"summary"`
	Externals     json.RawMessage `json:"external_bugs"`
	DateStamp     time.Time
	Age           int
}

// Bugs is a list of, well, bugs
type Bugs struct {
	Bugs []Bug `json:"bugs"`
}
