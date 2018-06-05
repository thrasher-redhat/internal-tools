// Entry point for the API to serve data
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	flag "github.com/spf13/pflag"

	"github.com/thrasher-redhat/internal-tools/pkg/api"
	"github.com/thrasher-redhat/internal-tools/pkg/assets"
	"github.com/thrasher-redhat/internal-tools/pkg/db"
	"github.com/thrasher-redhat/internal-tools/pkg/graphiql"
	"github.com/thrasher-redhat/internal-tools/pkg/options"
	"github.com/thrasher-redhat/internal-tools/pkg/resolverctx"
)

// graphiqlHandler is a simple handler to serve the GraphiQL interface.
func graphiqlHandler(w http.ResponseWriter, r *http.Request) {
	// Generated from graphiql.html with go-bindata like so:
	// go-bindata -pkg graphiql -o pkg/graphiql/page.go cmd/serve/graphiql.html
	bytes, err := graphiql.Asset("cmd/serve/graphiql.html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "404 - GraphiQL resource not found.")
	}
	fmt.Fprintf(w, string(bytes))
}

// Flags
var hostName = flag.StringP("hostname", "h", "postgresql", "the database hostname")
var configFile = flag.StringP("config", "c", "/etc/internal-tools/serve_cfg.yaml", "the configurations file")

func main() {
	flag.Parse()

	// Updates to the configmap should result in the pod restarting
	// So configs should always be reloaded on changes
	configs, err := options.PopulateConfigs(configFile)
	if err != nil {
		log.Fatalf("Unable to get configs: %v", err)
	}

	// Prepare the database connection, releases, then create the resolver
	db, err := db.NewClient(
		os.Getenv("POSTGRESQL_USER"),
		os.Getenv("POSTGRESQL_PASSWORD"),
		os.Getenv("POSTGRESQL_DATABASE"),
		"disable",
		*hostName,
	)
	if err != nil {
		log.Fatalf("Unable to create a database client: %v", err)
	}
	defer db.Close()

	resolver, err := api.NewResolver(configs.Releases, configs.Blockers)
	if err != nil {
		log.Fatalf("Unable to create resolver: %v", err)
	}

	// Uses go-bindata to generate the assets package from schema.graphql
	// go-bindata -pkg assets -o pkg/assets/schema.go pkg/api/schema.graphql
	// Loads and parses schema - will panic on error
	byteSchema := assets.MustAsset("pkg/api/schema.graphql")
	schema := graphql.MustParseSchema(string(byteSchema), resolver)

	// GraphiQL frontend for testing queries
	http.HandleFunc("/", graphiqlHandler)
	handler := &relay.Handler{Schema: schema}
	http.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tx, err := db.Begin()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "500 Error: Unable to complete request.")
			log.Printf("Unable to begin a transaction: %v", err)
			return
		}
		defer func() {
			if err := tx.End(); err != nil {
				log.Printf("Unable to commit transaction: %v", err)
			}
		}()
		ctx = resolverctx.WithTx(ctx, tx)
		r = r.WithContext(ctx)

		handler.ServeHTTP(w, r)
	})

	log.Println("LISTENING...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
