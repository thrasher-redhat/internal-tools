package api

import (
	"fmt"
	"log"
	"time"

	"github.com/thrasher-redhat/internal-tools/pkg/db"
	"github.com/thrasher-redhat/internal-tools/pkg/options"
)

// All dates should use the YYYY-MM-DD format
var dateFormat = "2006-01-02"

// Resolver represents the connection between the API and the data
type Resolver struct {
	dbClient db.Client
	releases map[string]options.Release
	blockers []string
}

// NewResolver is a factory for Resolver
func NewResolver(db db.Client, releases []options.Release, blockers []string) (*Resolver, error) {

	// Create a map of releases using the release name as the key
	m := make(map[string]options.Release)
	for _, release := range releases {
		m[release.Name] = release
	}

	return &Resolver{
		dbClient: db,
		releases: m,
		blockers: blockers,
	}, nil
}

// parseDatestamp checks for empty date or special strings before returning a date string.
func (r Resolver) parseDatestamp(datestamp string) (string, error) {
	// Default to latest
	switch datestamp {
	case "", "_latest":
		date, err := r.dbClient.GetLatest()
		if err != nil {
			return "", newAPISafeError(err, "Error retreiving latest date")
		}
		datestamp = date.Format(dateFormat)
	case "_earliest":
		date, err := r.dbClient.GetEarliest()
		if err != nil {
			return "", newAPISafeError(err, "Error retreiving earliest date")
		}
		datestamp = date.Format(dateFormat)
	}
	return datestamp, nil
}

// parseComponents checks for empty lists and works around the graphql
// listpacker bug.
func parseComponents(components *[]string) []string {
	// Note - If components are renamed in the future, we can check for it
	// and add an alias here to avoid updating the database

	// Workaround graphql's pointer to slice bug:
	// If you use a list that isn't []interface{}, then
	// Pack() uses a interface slice where the first element is given list
	// Current workaround is to use *[]type to get the actual array, so
	// dereference that pointer here to get the actual list.
	// https://github.com/graph-gophers/graphql-go/blob/b46637030579abd312c5eea21d36845b6e9e7ca4/internal/exec/packer/packer.go#L248-L250

	// If graphql-go gets updated, we should update the glide files and remove this
	var c []string
	if components != nil && len(*components) > 0 {
		c = *components
	} else {
		c = nil
	}
	return c
}

// getBugs queries the database for a list of bugs and converts to BugResolvers
func (r *Resolver) getBugs(datestamp string, components []string) ([]*BugResolver, error) {
	// Query the database
	bugs, err := r.dbClient.GetBugs(datestamp, components)
	if err != nil {
		return nil, newAPISafeError(err, "Error querying for list of bugs")
	}

	// Convert bugs into BugResolvers
	brs := make([]*BugResolver, len(bugs))
	for i, b := range bugs {
		brs[i] = &BugResolver{bug: b}
	}
	return brs, nil
}

// Bugs is a graphql query that fetches a list of bugs
func (r *Resolver) Bugs(args struct {
	Datestamp  string
	Components *[]string
}) ([]*BugResolver, error) {

	// Parse input
	date, err := r.parseDatestamp(args.Datestamp)
	if err != nil {
		safe, err := safeError(err, "Unable to parse date %q", args.Datestamp)
		log.Printf("Error parsing date: %v", err)
		return nil, safe
	}
	components := parseComponents(args.Components)

	// Query the database for a list of bugs given the datestamp/components)
	bugs, err := r.getBugs(date, components)
	if err != nil {
		safe, err := safeError(err, "Error getting list of bugs")
		log.Printf("Error querying for bugs: %v", err)
		return nil, safe
	}

	return bugs, nil
}

// Snapshot grabs the list of bugs and rollup for a given date
func (r *Resolver) Snapshot(args struct {
	Datestamp  string
	Components *[]string
}) (*SnapshotResolver, error) {

	// Parse input
	date, err := r.parseDatestamp(args.Datestamp)
	if err != nil {
		safe, err := safeError(err, "Unable to parse date %q", args.Datestamp)
		log.Printf("Error parsing date: %v", err)
		return nil, safe
	}
	components := parseComponents(args.Components)

	// Grab the list of bugs (as bugResolvers)
	brs, err := r.getBugs(date, components)
	if err != nil {
		safe, err := safeError(err, "Error getting list of bugs")
		log.Printf("Error querying for bugs: %v", err)
		return nil, safe
	}

	// Get the rollup for the current date
	// We will NOT filter by targetRelease for snapshot
	ru, err := r.getRollup(date, components, nil)
	if err != nil {
		safe, underlying := safeError(err, "Error getting rollup for date %q", date)
		log.Printf("Error getting snapshot rollup: %v", underlying)
		return nil, safe
	}

	//create and return a SnapshotResolver
	return &SnapshotResolver{
		datestamp: date,
		bugs:      brs,
		rollup:    ru,
	}, nil
}

// getRollup fetches date's totals for all (total), new, and closed bugs
// for each of all bugs, blocker bugs, and bugs with customer cases.
// Filters on components and targetRelease if provided.
// We assume that the inputs have already been parsed.
// Returns nil if the given datestamp has no bugs.
func (r *Resolver) getRollup(datestamp string, components []string, targets []string) (*RollupResolver, error) {
	// Get previous date to compare for new/closed bugs
	previousTime, err := r.dbClient.GetPreviousDate(datestamp)
	if err != nil {
		return nil, newAPISafeError(fmt.Errorf("unable to get previous date: %v", err), "Unable to create rollup for date %q.  Error getting prior date for new/closed comparisons.", datestamp)
	}
	previous := previousTime.Format(dateFormat)

	// Get the breakdowns for each set
	all, err := r.dbClient.GetBreakdown(previous, datestamp, components, nil, false, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdown: %v", err)
	}
	blockers, err := r.dbClient.GetBreakdown(previous, datestamp, components, r.blockers, false, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdown: %v", err)
	}
	custCases, err := r.dbClient.GetBreakdown(previous, datestamp, components, nil, true, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdown: %v", err)
	}

	return &RollupResolver{
		datestamp:     datestamp,
		all:           &BreakdownResolver{all},
		blockers:      &BreakdownResolver{blockers},
		customerCases: &BreakdownResolver{custCases},
	}, nil
}

// getRollups gets a list of rollups for every day between startDate and endDate
// Dates with no data will be skipped
// Input is assumed to be parsed
func (r Resolver) getRollups(startDate, endDate time.Time, components, targets []string) ([]*RollupResolver, error) {
	var rollups []*RollupResolver

	// Loop over days between start and end date (inclusive)
	for d := startDate; !d.Equal(endDate.AddDate(0, 0, 1)); d = d.AddDate(0, 0, 1) {
		rollup, err := r.getRollup(d.Format(dateFormat), components, targets)
		if err != nil || rollup == nil {
			// Don't hard error if an individual rollup fails or doesn't exist
			// Simply skip it and move on
			continue
		}
		rollups = append(rollups, rollup)
	}

	return rollups, nil
}

// getRelease is a helper function to get a release rollup for the given release/components
func (r Resolver) getRelease(name string, components []string) (*ReleaseResolver, error) {
	// Lookup release info by name
	thisRelease, ok := r.releases[name]
	if !ok {
		err := fmt.Errorf("resolver: cannot find release with name %q", name)
		return nil, newAPISafeError(err, "cannot find release with name %q", name)
	}

	// Parse the GA date.  It will default to latest date in the db.
	end, err := r.parseDatestamp(thisRelease.Dates.Ga)
	if err != nil {
		safe, err := safeError(err, "Unable to parse date %q", thisRelease.Dates.Ga)
		log.Printf("Error parsing date: %v", err)
		return nil, safe
	}
	endDate, err := time.Parse(dateFormat, end)
	if err != nil {
		return nil, newAPISafeError(err, "error parsing date %q", end)
	}

	// Parse the start date.  It will default to the first appearence of the release targets.
	// If that errors, it will default to 9 weeks (3 sprints) before the end date.
	// TODO - should this instead look for the first date that used the target(s)?
	// SELECT MIN(datestamp) FROM bugs WHERE targetRelease in ARRAY(targets);
	start := thisRelease.Dates.Start
	if start == "" {
		startTime, err := r.dbClient.GetEarliestDateForTargets(thisRelease.Targets)
		if err != nil {
			log.Printf("Unable to find earliest date for release %q. Using 3 sprints instead. Error: %v", name, err)
			start = endDate.AddDate(0, 0, -63).Format(dateFormat)
		} else {
			start = startTime.Format(dateFormat)
		}
	}
	start, err = r.parseDatestamp(start)
	if err != nil {
		safe, err := safeError(err, "Unable to parse date %q", start)
		log.Printf("Error parsing date: %v", err)
		return nil, safe
	}
	startDate, err := time.Parse(dateFormat, start)
	if err != nil {
		return nil, newAPISafeError(err, "error parsing date %q", start)
	}

	// Error if startDate > endDate
	// This can happen for an older release if...
	// There is no defined startDate
	// There is a defined endDate
	// And the data does not go back before the endDate
	// Best way to fix is to update the startDate or remove the release as it is too old.
	if endDate.Before(startDate) {
		err = fmt.Errorf("release %q: endDate %q cannot be before startDate %q", name, endDate, startDate)
		return nil, newAPISafeError(err, "invalid dates for release %q", name)
	}

	rollups, err := r.getRollups(startDate, endDate, components, thisRelease.Targets)
	if err != nil {
		return nil, fmt.Errorf("release: error getting list of rollups: %v", err)
	}

	return &ReleaseResolver{
		release: thisRelease,
		rollups: rollups,
	}, nil
}

// Release creates, populates, and returns a ReleaseResolver
func (r Resolver) Release(args struct {
	Name       string
	Components *[]string
}) (*ReleaseResolver, error) {

	// Parse input
	components := parseComponents(args.Components)

	release, err := r.getRelease(args.Name, components)
	if err != nil {
		safe, underlying := safeError(err, "Error querying for release %q", args.Name)
		log.Printf("Error getting release information: %v", underlying)
		return nil, safe
	}

	return release, nil
}

// Rollups creates and returns a list of rollups over the past 3 sprints (9 weeks).
func (r Resolver) Rollups(args struct{ Components *[]string }) ([]*RollupResolver, error) {
	// Parse input and setup dates
	components := parseComponents(args.Components)

	endDate, err := r.dbClient.GetLatest()
	if err != nil {
		safe, underlying := safeError(err, "Error getting 'latest' date")
		log.Printf("Unable to get latest date: %v", underlying)
		return nil, safe
	}

	// Start the graph data 9 weeks (3 sprints) before the end date.
	startDate := endDate.AddDate(0, 0, -63)

	rollups, err := r.getRollups(startDate, endDate, components, nil)
	if err != nil {
		safe, underlying := safeError(err, "Unable to get list of rollup data")
		log.Printf("Error getting rollups: %v", underlying)
		return nil, safe
	}

	return rollups, nil
}

// Releases is the query endpoint to return a list of all releases
func (r Resolver) Releases(args struct{ Components *[]string }) ([]*ReleaseResolver, error) {
	components := parseComponents(args.Components)

	var releaseResolvers []*ReleaseResolver
	for name := range r.releases {
		// Get the releases
		releaseResolver, err := r.getRelease(name, components)
		if err != nil {
			safe, underlying := safeError(err, "Unable to retreive release information for %q", name)
			log.Printf("Error retrieving release information for %q: %v", name, underlying)
			return nil, safe
		}
		releaseResolvers = append(releaseResolvers, releaseResolver)
	}

	return releaseResolvers, nil
}
