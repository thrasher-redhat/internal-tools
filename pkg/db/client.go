// Package db manages the postgresql database connection
package db

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/thrasher-redhat/internal-tools/pkg/bugzilla"
)

// WriteClient knows how to write to the database
type WriteClient interface {
	SnapshotBugzilla(bugzilla.Bugs) error
	//SnapshotTrello() will be here in the future
}

// ReadClient knows how to query the database atomically (read-only)
type ReadClient interface {
	Begin() (ReadQuerier, error)
}

// ReadQuerier knows how to query the database in a transaction (read-only)
type ReadQuerier interface {
	GetLatest() (time.Time, error)
	GetEarliest() (time.Time, error)
	GetPreviousDate(string) (time.Time, error)
	GetEarliestDateForTargets([]string) (time.Time, error)
	GetBreakdown(startDate string, endDate string, components []string, keywords []string, custCase bool, TargetReleases []string) (Breakdown, error)
	GetBreakdowns(components []string, keywords []string, custCase bool, targetReleases []string) (map[time.Time]Breakdown, error)
	GetBugs(datestamp string, components []string) ([]bugzilla.Bug, error)

	End() error
}

// Client knows how to connect and interact with the database
type Client interface {
	ReadClient
	WriteClient
	Close()
}

type postgresClient struct {
	database *sql.DB
}

type postgresQuerier struct {
	tx   *sql.Tx
	txMu sync.Mutex
}

// NewClient creates and opens a db connection
// It will be on the user to close this client
func NewClient(user, pass, name, mode, host string) (Client, error) {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s host=%s", user, pass, name, mode, host)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	return &postgresClient{
		database: db,
	}, nil
}

// Close will close the client's database connection
func (c postgresClient) Close() {
	c.database.Close()
}

// clearBugs will remove all bugs with the given datestamp
func clearBugs(tx *sql.Tx, t time.Time) error {
	// Delete all bugs with given datestamp
	result, err := tx.Exec(`DELETE FROM bugs WHERE datestamp = ($1)`, t)
	if err != nil {
		return fmt.Errorf("unable to delete bugs with date %v: %v", t, err)
	}
	total, err := result.RowsAffected()
	if err != nil {
		return err
	}

	log.Printf("Removed %d bugs for date: %v\n", total, t.Format("2006-01-02"))
	return nil
}

// insertBug processes and inserts (via a copy statement) a bug into the database
// Bugs are inserted with today's date
func insertBug(stmt *sql.Stmt, b bugzilla.Bug) error {
	// TODO - Look into reflection or gogenerate
	_, err := stmt.Exec(
		b.ID,
		b.Component,
		b.TargetRelease,
		b.AssignedTo,
		b.Status,
		b.Summary,
		pq.Array(b.Keywords),
		b.PmScore,
		string(b.Externals),
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("unable to insert bug with id %d: %v", b.ID, err)
	}

	return nil
}

// StoreBugs preps and stores all provided bugs in the given transaction
func storeBugs(tx *sql.Tx, bugs bugzilla.Bugs) error {
	// Copy is faster than insert for mass inserts like this
	// Similar to `INSERT INTO bugs(...) VALUES(...);` but faster under the hood
	stmt, err := tx.Prepare(pq.CopyIn("bugs",
		"id",
		"component",
		"target_release",
		"assigned_to",
		"status",
		"summary",
		"keywords",
		"cf_pm_score",
		"externals",
		"datestamp",
	))
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert each bug
	for _, bug := range bugs.Bugs {
		err = insertBug(stmt, bug)

		if err != nil {
			return err
		}
	}

	// Flushing buffered data
	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

// SnapshotBugzilla removes today's bugs (if any) and stores the new bugs in a single transaction
func (c *postgresClient) SnapshotBugzilla(bugs bugzilla.Bugs) error {
	// Setup transaction to remove today's bugs AND insert new bugs for today
	tx, err := c.database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear today's (old) bugs, if any
	err = clearBugs(tx, time.Now())
	if err != nil {
		log.Println("Error clearing bugs - rolling back snapshot process")
		return err
	}

	// Add today's (new) bugs
	err = storeBugs(tx, bugs)
	if err != nil {
		log.Println("Error storing bugs - rolling back snapshot process")
		return err
	}

	log.Println("Commiting transaction")
	return tx.Commit()
}
