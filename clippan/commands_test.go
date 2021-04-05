package clippan

import (
	"context"
	"testing"

	"github.com/iivvoo/clippan/bench"
	"github.com/stretchr/testify/assert"
)

func TestCommands(t *testing.T) {
	DB := bench.DBSession("test-commands")

	t.Run("Test `use` (success)", DB(func(cdb *bench.CouchDB, t *testing.T) {
		assert := assert.New(t)

		printer := &TestPrinter{}
		c := NewTestClippan(cdb, false, printer)
		testDB := "testing-commands-use"

		// Create testing database if it doesn't already exist
		exists, err := cdb.Client().DBExists(context.TODO(), testDB)
		assert.NoError(err)
		if !exists {
			assert.NoError(cdb.Client().CreateDB(context.TODO(), testDB))
		}

		c.Executer("use " + testDB)
		assert.Len(printer.Errors, 0)
	}))
	t.Run("Test `use` (fail)", DB(func(cdb *bench.CouchDB, t *testing.T) {
		assert := assert.New(t)

		printer := &TestPrinter{}
		c := NewTestClippan(cdb, false, printer)
		testDB := "testing-commands-use-doesnotexistLGAIEYGFIW"

		c.Executer("use " + testDB)

		assert.Len(printer.Errors, 1)
	}))
}
