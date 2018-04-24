package options

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// Milestones represent the important dates for a release
type Milestones struct {
	Start           string `yaml:"start"`
	FeatureComplete string `yaml:"feature_complete"`
	CodeFreeze      string `yaml:"code_freeze"`
	Ga              string `yaml:"ga"`
}

// Release represents the info needed to search/filter by release
type Release struct {
	Name string `yaml:"name"`
	// Targets will often only have the name, but could have more
	Targets []string `yaml:"targets"`
	// TODO - look into default bool and what its used for
	//Default bool `yaml: "default"`
	Dates Milestones `yaml:"milestones"`
}

// Configs is the top level of the given yaml file
type Configs struct {
	Releases []Release `yaml:"Releases"`
	Blockers []string  `yaml:"blockers"`
}

// PopulateConfigs reads the given yaml file and populates the configuration options structs
func PopulateConfigs(fileName *string) (Configs, error) {
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
