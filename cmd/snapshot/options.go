package main

import (
	"fmt"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// BugzillaConfigs stores the query and login information needed for the Bugzilla API
type BugzillaConfigs struct {
	Search  string   `yaml:"search"`
	ShareID string   `yaml:"sharer"`
	Fields  []string `yaml:"fields"`
	URL     string   `yaml:"url"`
	User    string   `yaml:"user"`
	Pass    string   `yaml:"pass"`
}

// SourceConfigs struct holds credentials for each API we need to access
type SourceConfigs struct {
	Bugzilla BugzillaConfigs `yaml:"bugzilla"`
	//Trello TrelloConfigs `yaml:"trello"`
}

// Configs holds all of the configuration objects
type Configs struct {
	Sources SourceConfigs `yaml:"Sources"`
}

// populateConfigs reads the given yaml file and populates the configuration options structs
func populateConfigs(fileName *string) (Configs, error) {
	file, err := os.Open(*fileName)
	if err != nil {
		return Configs{}, fmt.Errorf("unable to open configuration file: %v", err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return Configs{}, fmt.Errorf("unable to read configuration file: %v", err)
	}

	var c Configs
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return Configs{}, fmt.Errorf("cannot unmarshal yaml file: %v", err)
	}

	return c, nil
}
