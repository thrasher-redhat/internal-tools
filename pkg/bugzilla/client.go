package bugzilla

import (
	"bytes"
	"encoding/json"
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
	Id     uint64         `json:"id"`
}

// clientResponse is the JSONRPC wrapper structure for responses
type clientResponse struct {
	Id     uint64      `json:"id"`
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
	SharerId         string   `json:"sharer_id"`
	IncludeFields    []string `json:"include_fields"`
}

// ExecuteQuery returns all bugs that match the given saved query
func (bz *httpBugzillaClient) ExecuteQuery(query, sharer string, fields []string) (Bugs, error) {
	// Prepare the http request
	args := arguments{
		BugzillaLogin:    bz.username,
		BugzillaPassword: bz.password,
		SavedSearch:      query,
		SharerId:         sharer,
		IncludeFields:    fields,
	}

	req := &clientRequest{
		Method: "Bug.search",
		Params: [1]interface{}{args},
		Id:     0,
	}

	byteReq, err := json.Marshal(req)
	if err != nil {
		return Bugs{}, err
	}

	// Send the query over post and parse the response
	response, err := http.Post(bz.url, "application/json", bytes.NewReader(byteReq))
	if err != nil {
		return Bugs{}, err
	}
	defer response.Body.Close()

	byteRes, err := ioutil.ReadAll(response.Body)
	var results clientResponse
	err = json.Unmarshal(byteRes, &results)
	if err != nil {
		return Bugs{}, err
	}

	return results.Result, err
}
