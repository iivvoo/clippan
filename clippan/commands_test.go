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
		c := NewTestClippan(cdb, false, printer, NewMockEditor(), NewMockPrompt())
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
		c := NewTestClippan(cdb, false, printer, NewMockEditor(), NewMockPrompt())
		testDB := "testing-commands-use-doesnotexistLGAIEYGFIW"

		c.Executer("use " + testDB)

		assert.Len(printer.Errors, 1)
	}))
}

func TestEditPut(t *testing.T) {
	DB := bench.DBSession("test-edit-put")

	t.Run("Test normal put/create flow", DB(func(cdb *bench.CouchDB, t *testing.T) {
		assert := assert.New(t)
		printer := &TestPrinter{}

		json := []byte(`{"_id":"test1", "v":42}`)
		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			NewMockPrompt().SetMockData("a"),
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("put test1")
		assert.Len(printer.Errors, 0)

		var doc map[string]interface{}

		found, err := bench.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		assert.True(found)
		// standard unmarshalling makes it a float64, not int
		assert.EqualValues(42, doc["v"].(float64))
	}))

	t.Run("Test normal edit flow", DB(func(cdb *bench.CouchDB, t *testing.T) {
		assert := assert.New(t)
		printer := &TestPrinter{}

		json := []byte(`{"_id":"test1", "v":42}`)

		rev, err := cdb.DB().Put(context.TODO(), "test1", json)
		assert.NoError(err)

		// It should exist now, let's update
		json = []byte(`{"_id":"test1", "_rev": "` + rev + `", "v":43}`)

		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			NewMockPrompt().SetMockData("a"),
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("edit test1")
		assert.Len(printer.Errors, 0)

		var doc map[string]interface{}

		found, err := bench.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		assert.True(found)
		// standard unmarshalling makes it a float64, not int
		assert.EqualValues(43, doc["v"].(float64))
	}))

	t.Run("Test edit only if exists", DB(func(cdb *bench.CouchDB, t *testing.T) {
		assert := assert.New(t)
		printer := &TestPrinter{}

		// It should exist now, let's update
		json := []byte(`{"_id":"test1", "v":43}`)

		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			NewMockPrompt().SetMockData("a"),
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("edit test1")
		assert.Len(printer.Errors, 1)

		var doc map[string]interface{}

		found, err := bench.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		// It should not have been created
		assert.False(found)
	}))
}
