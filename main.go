package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/adlio/trello"
	_ "github.com/lib/pq"
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

// BugzillaCredentials stores the login information needed to access Bugzilla API
type BugzillaCredentials struct {
	URL  string `yaml:"url"` //"https://bugzilla.redhat.com/xmlrpc.cgi"
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

// DatabaseCredentials stores the login info and db info needed for the local postgresql instance
type DatabaseCredentials struct {
	User         string `yaml:"user"`
	Pass         string `yaml:"pass"`
	DatabaseName string `yaml:"dbname"`
	SslMode      string `yaml:"sslmode"`
}

// Credentials struct to hold all API access information
type Credentials struct {
	Trello   TrelloCredentials   `yaml:"trello"`
	Bugzilla BugzillaCredentials `yaml:"bugzilla"`
	Database DatabaseCredentials `yaml:"database"`
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
	//trelloTool(creds)

	// Bugzilla API Proof of Concepts
	//bugzillaTool(creds)
	//mbugzillaTool(creds)

	// PostgreSQL database
	//postgresqlTool(creds)

	// Combining bugzilla and database
	fmt.Println("\nGrabbing bugs from given query and inserting them into the database")
	mbugzillaTwo(creds)
	fmt.Println("\nReading database and printing all bugs that are not medium or low severity")
	readDB(creds)
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

	// Pretty print JSON
	var output bytes.Buffer
	err = json.Indent(&output, []byte(buglist), "", "\t")
	if err != nil {
		log.Fatalf("JSON parse error: %s\n", err)
	}
	fmt.Println(string(output.Bytes()))
}

// Function to grab list of bugs and upsert to database
func mbugzillaTwo(creds Credentials) {
	client := mbugzilla.NewClient(creds.Bugzilla.User, creds.Bugzilla.Pass, "https://bugzilla.redhat.com/xmlrpc.cgi")

	result, err := client.GetList("v2 Must Fix for Upcoming Release")
	if err != nil {
		log.Fatalf("Unable to get list: %s", err)
	}

	for _, bug := range result.Bugs {
		fmt.Println(bug.Id, bug.AssignedTo, bug.Priority, bug.Severity)
		err = addRow(creds, bug)
		if err != nil {
			log.Printf("Error inserting: %v\n", err)
		}
	}

}

// Basic functionality test for reading from a (local) postgreSQL database
func postgresqlTool(creds Credentials) {
	// This connection string can have...
	// user, password, dbname, host, port, sslmode, fallback_application_name, connect_timeout, sslcert, sllkey, sslrootcert
	// sslmode can be require, verify-full, verify-ca, or disable
	connStr := fmt.Sprintf("user=%s password = %s dbname=%s sslmode=%s", creds.Database.User, creds.Database.Pass, creds.Database.DatabaseName, creds.Database.SslMode)
	// db is intended to be long lived - open once and pass it to functions and then close it when done
	// so NOT how we're using it here...
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	// For prepared queries, it would look something like...
	// db.Query("SELECT id, name FROM users WHERE id = $1", id)
	rows, err := db.Query("SELECT version()")
	if err != nil {
		log.Fatalf("Error with query: %v", err)
	}
	defer rows.Close()

	var version string
	for rows.Next() {
		err = rows.Scan(&version)
		if err != nil {
			log.Printf("Error reading row: %v\n", err)
		}
		fmt.Println(version)
	}
	err = rows.Err()
	if err != nil {
		log.Fatalf("Error encountered during iteration: %v\n", err)
	}
}

// Adds a row to the database if one does not already exist with that id
func addRow(creds Credentials, bug mbugzilla.Bug) (err error) {
	// Setup database connections
	// This should really be passed along or set as a global
	connStr := fmt.Sprintf("user=%s password = %s dbname=%s sslmode=%s", creds.Database.User, creds.Database.Pass, creds.Database.DatabaseName, creds.Database.SslMode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO bugs(id, assigned, priority, severity) VALUES($1, $2, $3, $4)
						ON CONFLICT (id) DO NOTHING`, bug.Id, bug.AssignedTo, bug.Priority, bug.Severity)

	if err != nil {
		return err
	}

	return nil
}

func readDB(creds Credentials) {
	// Setup database connections
	connStr := fmt.Sprintf("user=%s password = %s dbname=%s sslmode=%s", creds.Database.User, creds.Database.Pass, creds.Database.DatabaseName, creds.Database.SslMode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer db.Close()

	result, err := db.Query(`SELECT * FROM bugs WHERE severity != 'low' AND severity != 'medium'`)
	if err != nil {
		log.Fatalf("Error on query: %v", err)
	}

	var id int
	var assigned, priority, severity string
	for result.Next() {
		err = result.Scan(&id, &assigned, &priority, &severity)
		if err != nil {
			log.Printf("Error reading row: %v\n", err)
		}
		fmt.Printf("Bug %d - Assigned to: %s\n", id, assigned)
		fmt.Printf("Priority: %s\nSeverity: %s\n----------\n", priority, severity)
	}
	err = result.Err()
	if err != nil {
		log.Fatalf("Error encountered during iteration: %v\n", err)
	}

}
