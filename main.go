package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/adlio/trello"
	mbugzilla "github.com/mfojtik/bugtraq/bugzilla"
	"github.com/mkorenkov/bugzilla"
	flag "github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v2"
)

// TrelloCredentials provides key and token to access Trello API
type TrelloCredentials struct {
	Key   string `yaml:"key"`
	Token string `yaml:"token"`
}

// BugzillaCredentials stores the login informaiton needed to access Bugzilla API
type BugzillaCredentials struct {
	URL  string `yaml:"url"` //"https://bugzilla.redhat.com/xmlrpc.cgi"
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

// Credentials struct to hold all API access information
type Credentials struct {
	Trello   TrelloCredentials   `yaml:"trello"`
	Bugzilla BugzillaCredentials `yaml:"bugzilla"`
}

var credsFile = flag.StringP("config", "c", "/etc/internal-tools/creds.yaml", "the credentials file")

func main() {
	fmt.Println("こんにちは世界")

	// Get the credentials
	// Will likely eventually use openshift env variables with os.Getenv()
	flag.Parse()

	file, err := os.Open(*credsFile)
	if err != nil {
		log.Fatalf("file stuff: %v", err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("error is %v", err)
	}

	var creds Credentials
	err = yaml.Unmarshal(data, &creds)
	if err != nil {
		log.Fatalf("Auth error: %v", err)
	}

	// Trello API Proof of Concept
	trelloTool(creds)

	// Bugzilla API Proof of Concepts
	bugzillaTool(creds)
	mbugzillaTool(creds)
}

func trelloTool(creds Credentials) {
	client := trello.NewClient(creds.Trello.Key, creds.Trello.Token)

	board, err := client.GetBoard("oBAYDsts", trello.Defaults())
	if err != nil {
		log.Fatalf("no boardsssssss: %v", err)
	}
	lists, err := board.GetLists(trello.Defaults())
	if err != nil {
		log.Fatalf("no listsssssssss: %v", err)
	}

	for _, list := range lists {
		cards, err := list.GetCards(trello.Defaults())
		if err != nil {
			log.Printf("Cannot get cards: %v", err)
			continue
		}
		fmt.Printf("List '%s' has %v cards\n", list.Name, len(cards))
		//for _, card := range cards {
		//	fmt.Printf("Name: %s\n", card.Name)
		//}
	}

}

// This library has a very narrow set of pre-defined calls
// Also is generic bugzilla, so can run afoul of red hat bugzilla quirks
// Likely would require taking the code as a base and then modifying and building on top of
func bugzillaTool(creds Credentials) {
	client, err := bugzilla.NewClient(creds.Bugzilla.URL, creds.Bugzilla.User, creds.Bugzilla.Pass)
	if err != nil {
		log.Fatalf("Unable to auth to bugzilla: %v", err)
	}

	v, err := client.BugzillaVersion()
	if err != nil {
		log.Fatalf("Unable to get bugzilla version: %v", err)
	}
	fmt.Printf("Bugzilla Version: %v\n", v)

	// BugList does an empty search...which the Red hat bugzilla doesn't allow
	// May need to fork this bugzilla API and do our own bug list function...
	//func (client *Client) OpenshiftBugList (limit int, offset int)([]Bug, error)

	/*bugs, err := client.BugList(10, 0)
	if err != nil {
		log.Fatalf("Unable to get list of bugs: %v", err)
	}
	for _, bug := range bugs {
		fmt.Println(bug.Assignee, bug.Subject)
	}*/
}

// Michal's implementation is limited to using named searches, but thats likely all we want (for now)
// It is Red Hat specific, which is nice
// We can just use the bugzilla package...but would the cache do anything for us?
func mbugzillaTool(creds Credentials) {
	//create RedHat with user, pass, listID
	//r.GetListJSON()
	r := mbugzilla.RedHat{
		User:     creds.Bugzilla.User,
		Password: creds.Bugzilla.Pass,
		ListId:   "v2 Must Fix for Upcoming Release",
	}
	buglist, err := r.GetListJSON()
	if err != nil {
		log.Fatalf("Unable to get list: %s", err)
	}
	fmt.Println(r.ListId)
	var output bytes.Buffer
	err = json.Indent(&output, []byte(buglist), "", "\t")
	if err != nil {
		log.Fatalf("JSON parse error: %s\n", err)
	}
	fmt.Println(string(output.Bytes()))
}
