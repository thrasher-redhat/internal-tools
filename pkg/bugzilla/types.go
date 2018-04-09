package bugzilla

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

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
	Status        string          `json:"status"`
	Summary       string          `json:"summary"`
	Component     SingleElemSlice `json:"component"`
	TargetRelease SingleElemSlice `json:"target_release"`
	AssignedTo    string          `json:"assigned_to"`
	Keywords      []string        `json:"keywords"`
	PmScore       Score           `json:"cf_pm_score"`
	Externals     json.RawMessage `json:"external_bugs"`
	DateStamp     time.Time
}

// Bugs is a list of, well, bugs
type Bugs struct {
	Bugs []Bug `json:"bugs"`
}
