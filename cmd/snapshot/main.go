// Entry point for snapshot process
package main

import (
	"log"

	flag "github.com/spf13/pflag"

	"github.com/thrasher-redhat/internal-tools/pkg/bugzilla"
	"github.com/thrasher-redhat/internal-tools/pkg/db"
)

var credsFile = flag.StringP("config", "c", "/etc/internal-tools/creds.yaml", "the credentials file")

func main() {
	// Grab the configuration values
	flag.Parse()
	configs, err := populate(credsFile)
	if err != nil {
		log.Fatalf("Unable to get configs: %v", err)
	}

	// Create a custom bugzilla client and fetch list of bugs
	bugClient := bugzilla.NewClient(
		configs.Sources.Bugzilla.User,
		configs.Sources.Bugzilla.Pass,
		configs.Sources.Bugzilla.URL,
	)
	bugs, err := bugClient.ExecuteQuery(
		configs.Sources.Bugzilla.Search,
		configs.Sources.Bugzilla.ShareID,
		configs.Sources.Bugzilla.Fields,
	)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	log.Printf("Query found %d bugs\n", len(bugs.Bugs))

	// Don't overwrite old bugs if we have no new bugs
	if len(bugs.Bugs) == 0 {
		log.Fatalf("Query found no bugs. Ensure query is correct.\n")
	}

	// Create database client and store the bugs
	dbClient, err := db.NewClient(
		configs.Sources.Database.User,
		configs.Sources.Database.Pass,
		configs.Sources.Database.DatabaseName,
		configs.Sources.Database.SslMode,
	)
	if err != nil {
		log.Fatalf("Error creating database client: %v", err)
	}
	defer dbClient.Close()
	err = dbClient.SnapshotBugzilla(bugs)
	if err != nil {
		log.Fatalf("Error storing snapshot to database: %v", err)
	}

	log.Println("Snapshot done.")
}
