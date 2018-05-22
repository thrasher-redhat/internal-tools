package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/thrasher-redhat/internal-tools/pkg/bugzilla"
)

// GetLatest provides the most recent datestamp in the database.
func (c postgresClient) GetLatest() (time.Time, error) {
	// Query the database to get the latest datestamp
	var t time.Time
	err := c.database.QueryRow("SELECT MAX(datestamp) FROM bugs").Scan(&t)

	if err != nil {
		return time.Time{}, fmt.Errorf("error scanning row for latest datestamp: %v", err)
	}

	return t, nil
}

// GetEarliest provides the oldest datestamp in the database.
func (c postgresClient) GetEarliest() (time.Time, error) {
	// Query the database to get the earliest datestamp
	var t time.Time
	err := c.database.QueryRow("SELECT MIN(datestamp) FROM bugs").Scan(&t)

	if err != nil {
		return time.Time{}, fmt.Errorf("error scanning row for earliest datestamp: %v", err)
	}

	return t, nil
}

// GetEarliestDateForTargets finds the first datestamp where the given target releases appeared
func (c postgresClient) GetEarliestDateForTargets(targets []string) (time.Time, error) {
	var t time.Time
	if targets == nil || len(targets) == 0 {
		// No targets provided
		return time.Time{}, fmt.Errorf("unable to get earliest date for targets: invalid targets %q", targets)
	}

	err := c.database.QueryRow("SELECT MIN(datestamp) FROM bugs WHERE target_release = ANY($1)", pq.Array(targets)).Scan(&t)

	if err != nil {
		return time.Time{}, fmt.Errorf("error scanning row for earliest date with targets %q: %v", targets, err)
	}

	return t, nil
}

// GetPreviousDate checks for the existence of the given date and then gets the snapshot preceding that given date.
// If the query is successful, but there are zero results, will return zerotime to be used as the previous date.
func (c postgresClient) GetPreviousDate(date string) (time.Time, error) {
	// TODO: Use a transaction...?

	// Check for the existence of given date.
	var ct int
	err := c.database.QueryRow("SELECT COUNT(id) FROM bugs WHERE datestamp = $1", date).Scan(&ct)
	if err != nil {
		return time.Time{}, err
	}
	if ct == 0 {
		return time.Time{}, fmt.Errorf("query: cannot get previous date as given date %q is not present", date)
	}

	// Query the database to get the date before the given date
	var t time.Time
	err = c.database.QueryRow("SELECT DISTINCT datestamp FROM bugs WHERE datestamp < $1 ORDER BY datestamp DESC LIMIT 1", date).Scan(&t)
	if err == sql.ErrNoRows {
		// The given date is the earliset possible date in the database; there are no datestamps before it
		log.Printf("Found no datestamps from before %q in the database: %v", date, err)
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, fmt.Errorf("error scanning row for previous datestamp: %v", err)
	}

	// If there is no valid previous date, it should return zerotime with no error
	return t, nil
}

// Put together a base query/set of parameters and a conditional/second set of parameters
// Conditional MUST only use a postgresql ARRAY[val1, val2, val3...] for parameters
// and should have a SINGLE appropriate format specifier %v
// Ex) "WHERE datestamp = %v" or "AND component = Any(ARRAY[%v])"
// Note that a space is added between the query and conditional
func appendQueryConditional(baseQuery string, baseArgs []interface{}, conditional string, conditionalArgs []interface{}) (string, []interface{}) {
	var query string
	var markers []string
	args := baseArgs
	// Account for 0 index and existing arguments
	floor := 1 + len(baseArgs)

	// Generate a slice of all the paramater markers to be inserted
	// Also add each element into the args slice as we iterate through
	for i, arg := range conditionalArgs {
		markers = append(markers, fmt.Sprintf("$%v", floor+i))
		args = append(args, arg)
	}
	query = baseQuery + fmt.Sprintf(" "+conditional, strings.Join(markers, ", "))

	return query, args
}

// Functions to query the database

// getBugs queries for a list of bugs
func (c postgresClient) GetBugs(datestamp string, components []string) ([]bugzilla.Bug, error) {
	// Base query
	// Grabs all components of the bug from bugs
	// Also grabs the "bug age", the difference between the given datestamp
	// and the MIN(datestamp) for that id (using a view)
	query := "SELECT bugs.id, bugs.component, bugs.target_release, bugs.assigned_to, bugs.status, bugs.summary, bugs.keywords, bugs.cf_pm_score, bugs.externals, bugs.datestamp, bugs.datestamp - bug_age.min FROM bugs, bug_age WHERE bugs.datestamp = $1 AND bugs.id = bug_age.id"
	args := []interface{}{datestamp}

	// Check if we need to filter by component and update query if so
	if components != nil {
		// Convert components to an interface slice
		typelessComponents := make([]interface{}, len(components))
		for i := range components {
			typelessComponents[i] = components[i]
		}
		// Append the query and arguments together
		query, args = appendQueryConditional(query, args, "AND bugs.component = Any(ARRAY[%v])", typelessComponents)
	}

	// Sort by pmScore
	query += " ORDER BY bugs.cf_pm_score DESC"

	rows, err := c.database.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Parse the data into bugzilla.Bugs objects
	var bugs []bugzilla.Bug

	for rows.Next() {
		var b bugzilla.Bug
		err = rows.Scan(
			&b.Id,
			&b.Component,
			&b.TargetRelease,
			&b.AssignedTo,
			&b.Status,
			&b.Summary,
			&b.Keywords,
			&b.PmScore,
			&b.Externals,
			&b.DateStamp,
			&b.Age,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
		} else {
			bugs = append(bugs, b)
		}
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error while scanning rows of bugs: %v", err)
	}

	return bugs, nil
}

// Breakdown represents the totals for all, new, and closed bugs in the query
type Breakdown struct {
	Total  int
	New    int
	Closed int
}

// GetBreakdown calculates the counts for total bugs, new bugs, and closed bugs
func (c postgresClient) GetBreakdown(startDate, endDate string, components []string, keywords []string, custCase bool, targetReleases []string) (Breakdown, error) {
	var total int
	var new int
	var closed int

	// Setup the base query
	query := "SELECT COUNT(id) FROM bugs WHERE datestamp = $1"
	args := []interface{}{endDate}

	// QUERY CRAFTING
	if components != nil {
		// Convert components to an interface slice
		typelessComponents := make([]interface{}, len(components))
		for i := range components {
			typelessComponents[i] = components[i]
		}
		// Append the query and arguments together
		query, args = appendQueryConditional(query, args, "AND component = Any(ARRAY[%v])", typelessComponents)
	}
	if len(keywords) > 0 {
		// Convert keywords to an interface slice
		typelessKeywords := make([]interface{}, len(keywords))
		for i := range keywords {
			typelessKeywords[i] = keywords[i]
		}
		// Append the query and arguments together
		query, args = appendQueryConditional(query, args, "AND keywords && ARRAY[%v]", typelessKeywords)
	}
	if custCase == true {
		// Query the jsonb directly
		// We care about external bz sources with the id that matches the "Red Hat Customer Portal"
		query, args = appendQueryConditional(query, args, "AND %v in (SELECT CAST( jsonb_array_elements(bugs.externals)->>'ext_bz_id' AS INT))", []interface{}{bugzilla.ExternalID})
	}
	if targetReleases != nil && len(targetReleases) > 0 {
		// Convert keywords to an interface slice
		typelessTargets := make([]interface{}, len(targetReleases))
		for i := range targetReleases {
			typelessTargets[i] = targetReleases[i]
		}
		query, args = appendQueryConditional(query, args, "AND target_release = Any(ARRAY[%v])", typelessTargets)
	}

	// Get the TOTAL number of bugs on the given day
	err := c.database.QueryRow(query, args...).Scan(&total)
	if err != nil {
		log.Printf("Error querying for TOTAL bug count: %v", err)
		return Breakdown{}, err
	}

	// Get the number of bugs that are NEW on the given day
	// We use the query for TOTAL and check against all bugs that existed on the previous date
	subQuery := "AND id NOT IN (SELECT id FROM bugs WHERE datestamp = %v)"
	newQuery, newArgs := appendQueryConditional(query, args, subQuery, []interface{}{startDate})

	err = c.database.QueryRow(newQuery, newArgs...).Scan(&new)
	if err != nil {
		log.Printf("Error querying for NEW bug count: %v", err)
		return Breakdown{}, err
	}

	// See which of the bugs from "yesterday" are not in "today's" bug list
	// Uses the same query EXCEPT that the first element in closedArgs is startDate instead of endDate
	closedQuery, closedArgs := appendQueryConditional(query, args, "AND id NOT IN (SELECT id FROM bugs where datestamp = %v)", []interface{}{endDate})
	closedArgs[0] = startDate

	err = c.database.QueryRow(closedQuery, closedArgs...).Scan(&closed)
	if err != nil {
		log.Printf("Error querying for CLOSED bug count: %v", err)
		return Breakdown{}, err
	}

	return Breakdown{
		Total:  total,
		New:    new,
		Closed: closed,
	}, nil
}
