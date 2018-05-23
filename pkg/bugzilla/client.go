package bugzilla

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client knows how to execute a query for a saved search
// It represents a client for a bugzilla API
type Client interface {
	ExecuteQuery(query, sharer string, fields []string) (Bugs, error)
}

// httpBugzillaClient is a client for the bugzilla API that connects via JSONRPC over HTTP
type httpBugzillaClient struct {
	username string
	password string
	url      string
}

// clientRequest is the JSONRPC wrapper structure for requests
type clientRequest struct {
	Method string         `json:"method"`
	Params [1]interface{} `json:"params"`
	ID     uint64         `json:"id"`
}

// clientResponse is the JSONRPC wrapper structure for responses
type clientResponse struct {
	ID     uint64      `json:"id"`
	Result Bugs        `json:"result"`
	Error  interface{} `json:"error"`
}

// NewClient creates and returns a client
func NewClient(user, pass, address string) Client {
	return &httpBugzillaClient{
		username: user,
		password: pass,
		url:      address,
	}
}

// arguments is the set of arguments for a "savedsearch" query with the bugzilla RPC
type arguments struct {
	BugzillaLogin    string   `json:"Bugzilla_login"`
	BugzillaPassword string   `json:"Bugzilla_password"`
	SavedSearch      string   `json:"savedsearch"`
	SharerID         string   `json:"sharer_id"`
	IncludeFields    []string `json:"include_fields"`
}

// ExecuteQuery returns all bugs that match the given saved query
func (bz *httpBugzillaClient) ExecuteQuery(query, sharer string, fields []string) (Bugs, error) {
	// Prepare the http request
	args := arguments{
		BugzillaLogin:    bz.username,
		BugzillaPassword: bz.password,
		SavedSearch:      query,
		SharerID:         sharer,
		IncludeFields:    fields,
	}

	req := &clientRequest{
		Method: "Bug.search",
		Params: [1]interface{}{args},
		ID:     0,
	}

	byteReq, err := json.Marshal(req)
	if err != nil {
		return Bugs{}, fmt.Errorf("unable to marshal http request: %v", err)
	}

	// Send the query over post and parse the response
	response, err := http.Post(bz.url, "application/json", bytes.NewReader(byteReq))
	if err != nil {
		return Bugs{}, fmt.Errorf("error when POSTing query: %v", err)
	}
	defer response.Body.Close()

	byteRes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return Bugs{}, fmt.Errorf("error reading from response body: %v", err)
	}

	var results clientResponse
	err = json.Unmarshal(byteRes, &results)
	if err != nil {
		return Bugs{}, fmt.Errorf("unable to unmarshal http response: %v", err)
	}

	return results.Result, nil
}
