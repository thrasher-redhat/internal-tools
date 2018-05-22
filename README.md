# internal-tools
Initial testing and research for some internal metrics tools.

Uses the Red Hat Bugzilla API to grab data from a bugzilla saved query.  In the future, may also grab relevant trello information.  All information is stored into a postgresql database.  The data is snapshotted hourly, but previous snapshots for that day are removed - resulting in a single snapshot per day.

There will be an API server to access the data and information calculated from the data.

Ther will also be a frontend to display information and to visually represent some of the analytics.

This app is designed to be containerized and run on OpenShift.

## OpenShift Setup

NOTE: The OpenShift setup is a work in progress.  This repo will be updated with templates, s2i, and webhooks as time goes on.

Assuming you already have `oc` installed and have an OpenShift project...

### Postgresql Container

Setup a standard Postgresql template.  This will include a the pod, a persistent volume for storage, and a service to access the database.  It will generate a database name, user, and password, as well as setting the POSTGRESQL_DATABASE, POSTGRESQL_USER, and POSTGRESQL_PASSWORD environment variables.

There is some initial setup needed, so `oc rsh` into the postgresql pod.  Connect to the database with `psql` as the generated user.

    psql --dbname=$POSTGRESQL_DATABASE --username=$POSTGRESQL_USER

We can then initialize the database with the `database/*.sql` files.  Copy and paste the commands into psql to create the necessary tables and views.  You can leave psql with `\q`.

### Snapshoter

Create a docker image of the snapshot program.

    go build -o deploy/snapshot cmd/snapshot/*.go

    docker build -t snapshot:dev -f deploy/Dockerfile.snapshot deploy/

Or use `make` and `make images` (builds images for both snapshoter and server).

Apply the yaml file to create a Cron Job.

    oc apply -f deploy/snapshoter.yaml

Create your snapshot_cfg.yaml file from the template with the proper information and create a configmap to make it accessable to the pod.  The config map should be named 'snapshot-cfg' with a key of 'snapshot_cfg.yaml' and the contents of the file will be the value.

### Server

Create a docker image of the snapshot program.

    go build -o deploy/serve cmd/serve/*.go

    docker build -t serve:dev -f deploy/Dockerfile.serve deploy/

Or use `make` and `make images` (builds images for both snapshoter and server).

Apply the yaml files to create a Deployment, Service, and Route.  NOTE: These will likely be a single template in the future

    oc apply -f deploy/server.yaml
    oc apply -f deploy/server-service.yaml
    oc apply -f deploy/server-route.yaml

Populate the serve_cfg.yaml file with the proper information and add a configmap  to make it accessable to the pod.


## Local Setup

Alternatively, this program can be run locally for testing or dev purposes.

### Postgresql Database

Install and setup a postgresql database.  Create the necessary tables and views by running the `database/*.sql` files.

Export your postgresql username, password, and database name as the appropriate environment variables.

    export POSTGRESQL_USER="myusername"
    export POSTGRESQL_PASSWORD="mypassword"
    export POSTGRESQL_DATABASE="mydatabasename"

### Snapshoter

The snapshoter can be run as go code.  Use the --config (-c) and --hostname (-h) flags to pass in the location of the snapshot config yaml file and the database hostname (probably "localhost" if running locally).  Ensure the local database environement variables have been set up.

    go run cmd/snapshot/main.go cmd/snapshot/options.go -c /path/to/snapshot_cfg.yaml -h "localhost"

This can also be built and run as a cron job to more accurately replicate the data collection process.  The containerized version runs once an hour from 0600 to 2000.

### Server

The server can be run at localhost:8080.  Use the --config (-c) and --hostname (-h) flags to pass in the location of the server config yaml file and the database hostname (probably "localhost" if running locally).  Ensure the local database environement variables have been set up.

    go run cmd/serve/main.go -c ~/path/to/serve_cfg.yaml -h localhost

## Development

### Make

The `Makefile` contains a couple of commands.

* `make` or `make all` will build run `go build` for the snapshot and serve packages
* `make images` will `docker build` from the go executables
* `make all images` will do both in order

### Generate with go-bindata

The assets package is generated from the graphql schema using go-bindata.  If you wish to edit pkg/api/schema.graphql , then you'll need to re-generate the assets package.  If you don't have the `go-bindata` command, use `go get` to grab the go-bindata package and ensure your $PATH contains $GOPATH/bin or that the go-bindata executable is otherwise in your $PATH.

```
go get github.com/go-bindata/go-bindata
go-bindata -pkg assets -o pkg/assets/schema.go pkg/api/schema.graphql
```

Likewise, if you change the api endpoint, you may want to update graphiql.html to reflect that and then regenerate the graphiql package with the following command (assuming you already have go-bindata).  

```
go-bindata -pkg graphiql -o pkg/graphiql/page.go cmd/serve/graphiql.html
```

## License

Licensed under the MIT License.  See LICENSE file for more information.