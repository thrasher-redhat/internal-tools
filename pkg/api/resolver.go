package api

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/thrasher-redhat/internal-tools/pkg/db"
	"github.com/thrasher-redhat/internal-tools/pkg/options"
	"github.com/thrasher-redhat/internal-tools/pkg/resolverctx"
)

// All dates should use the YYYY-MM-DD format
var dateFormat = "2006-01-02"

// Resolver represents the connection between the API and the data
type Resolver struct {
	releases map[string]options.Release
	blockers []string
}

// NewResolver is a factory for Resolver
func NewResolver(releases []options.Release, blockers []string) (*Resolver, error) {

	// Create a map of releases using the release name as the key
	m := make(map[string]options.Release)
	for _, release := range releases {
		m[release.Name] = release
	}

	return &Resolver{
		releases: m,
		blockers: blockers,
	}, nil
}

// parseDatestamp checks for empty date or special strings before returning a date string.
func (r *Resolver) parseDatestamp(tx db.ReadQuerier, datestamp string) (string, error) {
	// Default to latest
	switch datestamp {
	case "", "_latest":
		date, err := tx.GetLatest()
		if err != nil {
			return "", newAPISafeError(err, "Error retreiving latest date")
		}
		datestamp = date.Format(dateFormat)
	case "_earliest":
		date, err := tx.GetEarliest()
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
func (r *Resolver) getBugs(tx db.ReadQuerier, datestamp string, components []string) ([]*BugResolver, error) {
	// Query the database
	bugs, err := tx.GetBugs(datestamp, components)
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
func (r *Resolver) Bugs(ctx context.Context, args struct {
	Datestamp  string
	Components *[]string
}) ([]*BugResolver, error) {

	tx, ok := resolverctx.GetTx(ctx)
	if !ok {
		log.Printf("Error getting transaction from context.")
		return nil, fmt.Errorf("Unable to get transaction from context")
	}

	// Parse input
	date, err := r.parseDatestamp(tx, args.Datestamp)
	if err != nil {
		safe, err := safeError(err, "Unable to parse date %q", args.Datestamp)
		log.Printf("Error parsing date: %v", err)
		return nil, safe
	}
	components := parseComponents(args.Components)

	// Query the database for a list of bugs given the datestamp/components)
	bugs, err := r.getBugs(tx, date, components)
	if err != nil {
		safe, err := safeError(err, "Error getting list of bugs")
		log.Printf("Error querying for bugs: %v", err)
		return nil, safe
	}

	return bugs, nil
}

// Snapshot grabs the list of bugs and rollup for a given date
func (r *Resolver) Snapshot(ctx context.Context, args struct {
	Datestamp  string
	Components *[]string
}) (*SnapshotResolver, error) {

	tx, ok := resolverctx.GetTx(ctx)
	if !ok {
		log.Printf("Error getting transaction from context.")
		return nil, fmt.Errorf("Unable to get transaction from context")
	}

	// Parse input
	date, err := r.parseDatestamp(tx, args.Datestamp)
	if err != nil {
		safe, err := safeError(err, "Unable to parse date %q", args.Datestamp)
		log.Printf("Error parsing date: %v", err)
		return nil, safe
	}
	components := parseComponents(args.Components)

	// Grab the list of bugs (as bugResolvers)
	brs, err := r.getBugs(tx, date, components)
	if err != nil {
		safe, err := safeError(err, "Error getting list of bugs")
		log.Printf("Error querying for bugs: %v", err)
		return nil, safe
	}

	// Get the rollup for the current date
	// We will NOT filter by targetRelease for snapshot
	ru, err := r.getRollup(tx, date, components, nil)
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

// getRollup fetches a single date's totals for all (total), new, and closed bugs
// for each of all bugs, blocker bugs, and bugs with customer cases.
// Filters on components and targetRelease if provided.
// We assume that the inputs have already been parsed.
// Returns nil if the given datestamp has no bugs.
// NOTE - Currently not used, but will remain in case a single rollup endpoint is added
func (r *Resolver) getRollup(tx db.ReadQuerier, datestamp string, components []string, targets []string) (*RollupResolver, error) {
	// Get previous date to compare for new/closed bugs
	previousTime, err := tx.GetPreviousDate(datestamp)
	if err != nil {
		return nil, newAPISafeError(fmt.Errorf("unable to get previous date: %v", err), "Unable to create rollup for date %q.  Error getting prior date for new/closed comparisons.", datestamp)
	}
	previous := previousTime.Format(dateFormat)

	// Get the breakdowns for each set
	all, err := tx.GetBreakdown(previous, datestamp, components, nil, false, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdown: %v", err)
	}
	blockers, err := tx.GetBreakdown(previous, datestamp, components, r.blockers, false, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdown: %v", err)
	}
	custCases, err := tx.GetBreakdown(previous, datestamp, components, nil, true, targets)
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
func (r *Resolver) getRollups(tx db.ReadQuerier, startDate, endDate time.Time, components, targets []string) ([]*RollupResolver, error) {
	var rollups []*RollupResolver

	// Query db for breakdown maps
	all, err := tx.GetBreakdowns(components, nil, false, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdowns for all bugs: %v", err)
	}
	blockers, err := tx.GetBreakdowns(components, r.blockers, false, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdowns for blocker bugs: %v", err)
	}
	custCases, err := tx.GetBreakdowns(components, nil, true, targets)
	if err != nil {
		return nil, fmt.Errorf("unable to get breakdowns for bugs with customer cases: %v", err)
	}

	// Loop through startDate to endDate (inclusive) and attempt to use d as key
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		var b db.Breakdown

		// Grab breakdowns and create BreakdownResolvers for to craft a rollup
		b, ok := all[d]
		if !ok {
			// If no data for given date, skip this date
			continue
		}
		tempAll := &BreakdownResolver{b}

		b, ok = blockers[d]
		if !ok {
			// If no data for given date, skip this date
			continue
		}
		tempBlockers := &BreakdownResolver{b}

		b, ok = custCases[d]
		if !ok {
			// If no data for given date, skip this date
			continue
		}
		tempCustCases := &BreakdownResolver{b}

		rollups = append(rollups, &RollupResolver{
			datestamp:     d.Format(dateFormat),
			all:           tempAll,
			blockers:      tempBlockers,
			customerCases: tempCustCases,
		})
	}

	return rollups, nil
}

// getRelease is a helper function to get a release rollup for the given release/components
func (r *Resolver) getRelease(tx db.ReadQuerier, name string, components []string) (*ReleaseResolver, error) {
	// Lookup release info by name
	thisRelease, ok := r.releases[name]
	if !ok {
		err := fmt.Errorf("resolver: cannot find release with name %q", name)
		return nil, newAPISafeError(err, "cannot find release with name %q", name)
	}

	// Parse the GA date.  It will default to latest date in the db.
	end, err := r.parseDatestamp(tx, thisRelease.Dates.Ga)
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
		startTime, err := tx.GetEarliestDateForTargets(thisRelease.Targets)
		if err != nil {
			log.Printf("Unable to find earliest date for release %q. Using 3 sprints instead. Error: %v", name, err)
			start = endDate.AddDate(0, 0, -63).Format(dateFormat)
		} else {
			start = startTime.Format(dateFormat)
		}
	}
	start, err = r.parseDatestamp(tx, start)
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

	rollups, err := r.getRollups(tx, startDate, endDate, components, thisRelease.Targets)
	if err != nil {
		return nil, fmt.Errorf("release: error getting list of rollups: %v", err)
	}

	return &ReleaseResolver{
		release: thisRelease,
		rollups: rollups,
	}, nil
}

// Release creates, populates, and returns a ReleaseResolver
func (r *Resolver) Release(ctx context.Context, args struct {
	Name       string
	Components *[]string
}) (*ReleaseResolver, error) {

	tx, ok := resolverctx.GetTx(ctx)
	if !ok {
		log.Printf("Error getting transaction from context.")
		return nil, fmt.Errorf("Unable to get transaction from context")
	}

	// Parse input
	components := parseComponents(args.Components)

	release, err := r.getRelease(tx, args.Name, components)
	if err != nil {
		safe, underlying := safeError(err, "Error querying for release %q", args.Name)
		log.Printf("Error getting release information: %v", underlying)
		return nil, safe
	}

	return release, nil
}

// Rollups creates and returns a list of rollups over the past 3 sprints (9 weeks).
func (r *Resolver) Rollups(ctx context.Context, args struct{ Components *[]string }) ([]*RollupResolver, error) {

	tx, ok := resolverctx.GetTx(ctx)
	if !ok {
		log.Printf("Error getting transaction from context.")
		return nil, fmt.Errorf("Unable to get transaction from context")
	}

	// Parse input and setup dates
	components := parseComponents(args.Components)

	endDate, err := tx.GetLatest()
	if err != nil {
		safe, underlying := safeError(err, "Error getting 'latest' date")
		log.Printf("Unable to get latest date: %v", underlying)
		return nil, safe
	}

	// Start the graph data 9 weeks (3 sprints) before the end date.
	startDate := endDate.AddDate(0, 0, -63)

	rollups, err := r.getRollups(tx, startDate, endDate, components, nil)
	if err != nil {
		safe, underlying := safeError(err, "Unable to get list of rollup data")
		log.Printf("Error getting rollups: %v", underlying)
		return nil, safe
	}

	return rollups, nil
}

// Releases is the query endpoint to return a list of all releases
func (r *Resolver) Releases(ctx context.Context, args struct{ Components *[]string }) ([]*ReleaseResolver, error) {

	tx, ok := resolverctx.GetTx(ctx)
	if !ok {
		log.Printf("Error getting transaction from context.")
		return nil, fmt.Errorf("Unable to get transaction from context")
	}

	components := parseComponents(args.Components)

	var releaseResolvers []*ReleaseResolver
	for name := range r.releases {
		// Get the releases
		releaseResolver, err := r.getRelease(tx, name, components)
		if err != nil {
			safe, underlying := safeError(err, "Unable to retreive release information for %q", name)
			log.Printf("Error retrieving release information for %q: %v", name, underlying)
			return nil, safe
		}
		releaseResolvers = append(releaseResolvers, releaseResolver)
	}

	return releaseResolvers, nil
}
