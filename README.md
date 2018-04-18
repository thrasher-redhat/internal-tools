# internal-tools
Initial testing and research for some internal metrics tools.

Uses the Red Hat Bugzilla API to grab data from a supplied bugzilla saved query.  In the future, will also grab relevant trello information.  All information is stored into a postgresql database.  The data is snapshotted hourly, but previous snapshots for that day are removed - resulting in a single snapshot per day.

Will generate and serve visual representations of the stored information.

This app is designed to be containerized and run on OpenShift.

## OpenShift Setup

NOTE: The OpenShift setup is a work in progress.  This repo will be updated with templates, s2i, and webhooks as time goes on.

Assuming you already have `oc` installed and have an OpenShift project...

### Postgresql Container

Setup a standard Postgresql template.  This will include a the pod, a persistent volume for storage, and a service to access the database.

Once running, the database will need to be initialized with the `database/*.sql` files, which will create the necessary tables.

### Snapshoter

Create a docker image of the snapshot program.

    go build -o deploy/snapshot cmd/snapshot/*.go

    docker build -t snapshot:dev -f deploy/Dockerfile.snapshot deploy/

Or use `make` and `make images` (builds images for both snapshoter and server).

Apply the yaml file to create a Cron Job.

    oc apply -f deploy/snapshoter.yaml

Populate the snapshot_cfg.yaml file with the proper information and add a configmap  to make it accessable to the pod.

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

Install and setup a postgresql database.  Create the necessary tables by running the `database/*.sql` files.

Export your postgresql username, password, and database name as the appropriate environment variables.

    export POSTGRESQL_USER="myusername"
    export POSTGRESQL_PASSWORD="mypassword"
    export POSTGRESQL_DATABASE="mydatabasename"

### Snapshoter

The snapshoter can be run as go code.  Use the --config (-c) and --hostname (-h) flags to pass in the location of the snapshot config yaml file and the database hostname (probably "localhost" if running locally).

    go run cmd/snapshot/main.go cmd/snapshot/options.go -c /path/to/snapshot_cfg.yaml -h "localhost"

This can also be built and run as a cron job to more accurately replicate the data collection process.  The containerized version runs once an hour from 0600 to 2000.

### Server

The server is currently very simple and can be run with:

    go run cmd/serve/main.go

In the future, the server will require access to a server_config.yaml as well as the postgresql information.  More information on those flags and environment variables will be added when appropriate.

## License

Licensed under the MIT License.  See LICENSE file for more information.