package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/adlio/trello"
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
	URL  string `yaml:"url"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

// Credentials struct to hold all API access information
type Credentials struct {
	Trello TrelloCredentials `yaml:"trello"`
}

var credsFile = flag.StringP("config", "c", "/etc/internal-tools/creds.yaml", "the credentials file")

func main() {
	fmt.Println("こんにちは世界")

	// Get the credentials
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
		log.Fatalf("shit.  Bloody authentication: %v", err)
	}

	// Trello API Proof of Concept
	trelloTool(creds)

	// Bugzilla API Proof of Concept

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
			log.Printf("oh shit: %v", err)
			continue
		}
		for _, card := range cards {
			fmt.Printf("Name: %s\n", card.Name)
		}
	}

}

func bugzillaTool(creds Credentials) {
	// Do bugzilla stuff here...
	fmt.Println("Nothing here yet...")
	return
}
