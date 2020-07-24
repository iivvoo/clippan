package bench

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-kivik/kivik/v4"
)

func DBSession(postfix string) func(func(*CouchDB, *testing.T)) func(*testing.T) {
	dsn := os.Getenv("COUCHDB_TESTING_DATABASE")

	// hardcode the test database to "Testing", don't take it from the env
	// just to make sure we don't accidentally test against a production database
	if dsn == "" {
		dsn = "http://admin:a-secret@localhost:5984"
	}
	database := "testing" + "-" + postfix

	u, err := url.Parse(dsn)
	if err != nil {
		return nil
	}
	u.Path = database
	dsn = u.String()

	cdb, err := NewCouchDB(dsn)
	if err != nil {
		panic(err)
	}

	return func(f func(db *CouchDB, t *testing.T)) func(*testing.T) {
		return func(t *testing.T) {
			cdb.Init()
			if exists, err := cdb.client.DBExists(context.TODO(), database); err == nil && exists {
				if err := cdb.client.DestroyDB(context.TODO(), database); err != nil {
					panic(err)
				}
			} else if err != nil {
				if c := kivik.StatusCode(err); c >= http.StatusBadGateway && c <= http.StatusGatewayTimeout {
					panic("Database could not be reached")
				}
				panic(err)
			}
			for i := 0; i < 5; i++ {
				if err := cdb.client.CreateDB(context.TODO(), database); err != nil {
					if kivik.StatusCode(err) == http.StatusPreconditionFailed {
						time.Sleep(time.Millisecond * 200)
						continue
					}
					panic(err)
				}
				break
			}

			cdb.db = cdb.client.DB(context.TODO(), database)

			// setup designdocs
			// Check(cdb.db) XXX

			// This does not cleanup after the test (only before a test runs) which allows
			// you to inspect the database after running a single test
			f(cdb, t)
		}
	}
}
