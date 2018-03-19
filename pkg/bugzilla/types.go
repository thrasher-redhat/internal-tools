package bugzilla

// Bug maps to the desired fields of a bugzilla bug
type Bug struct {
	Id            int      `json:"id"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	Product       string   `json:"product"`
	Component     []string `json:"component"`
	TargetRelease []string `json:"target_release"`
	AssignedTo    string   `json:"assigned_to"`
	Keywords      []string `json:"keywords"`
	PmScore       int      `json:"cf_pm_score"`
	Externals     []string `json:"external_bugs"`
}

// Bugs is a list of, well, bugs
type Bugs struct {
	Bugs []Bug `json:"bugs"`
}
