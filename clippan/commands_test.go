package clippan

import (
	"context"
	"testing"

	"github.com/iivvoo/clippan/helpers"
	"github.com/stretchr/testify/assert"
)

func TestCommands(t *testing.T) {
	DB := helpers.DBSession("test-commands")

	t.Run("Test `use` (success)", DB(func(cdb *helpers.CouchDB, t *testing.T) {
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
	t.Run("Test `use` (fail)", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)

		printer := &TestPrinter{}
		c := NewTestClippan(cdb, false, printer, NewMockEditor(), NewMockPrompt())
		testDB := "testing-commands-use-doesnotexistLGAIEYGFIW"

		c.Executer("use " + testDB)

		assert.Len(printer.Errors, 1)
	}))
}

func TestEditPut(t *testing.T) {
	DB := helpers.DBSession("test-edit-put")

	t.Run("Test normal put/create flow", DB(func(cdb *helpers.CouchDB, t *testing.T) {
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

		found, err := helpers.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		assert.True(found)
		// standard unmarshalling makes it a float64, not int
		assert.EqualValues(42, doc["v"].(float64))
	}))
	t.Run("Test invalid json put/create flow", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)
		printer := &TestPrinter{}

		json := []byte(`{"_id":"test1" "v":42}`) // , missing
		prompt := NewMockPrompt().SetMockData("a")
		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			prompt,
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("put test1")
		// Assert that input has been asked
		assert.Len(prompt.Inputs, 1)

		var doc map[string]interface{}

		found, err := helpers.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		assert.False(found)
	}))

	t.Run("Test normal edit flow", DB(func(cdb *helpers.CouchDB, t *testing.T) {
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

		found, err := helpers.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		assert.True(found)
		// standard unmarshalling makes it a float64, not int
		assert.EqualValues(43, doc["v"].(float64))
	}))

	t.Run("Test edit only if exists", DB(func(cdb *helpers.CouchDB, t *testing.T) {
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

		found, err := helpers.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		// It should not have been created
		assert.False(found)
	}))

	t.Run("Test conflict edit flow", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)
		printer := &TestPrinter{}

		json := []byte(`{"_id":"test1", "v":42}`)

		_, err := cdb.DB().Put(context.TODO(), "test1", json)
		assert.NoError(err)

		editor := &ConflictEditor{cdb: cdb, id: "test1"}
		prompt := NewMockPrompt().SetMockData("a")
		c := NewTestClippan(cdb,
			true,
			printer,
			editor,
			prompt,
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("edit test1")
		// Assert that input has been asked
		assert.Len(prompt.Inputs, 1)

		var doc map[string]interface{}

		found, err := helpers.GetOr404(cdb.GetDB(), "test1", &doc)
		assert.NoError(err)
		assert.True(found)
		// It shouldn't have changed
		assert.EqualValues(editor.revResult, doc["_rev"].(string))
		assert.EqualValues(42, doc["v"].(float64))
	}))

	// force not yet implemented
	// t.Run("Test conflict edit flow, force change", DB(func(cdb *helpers.CouchDB, t *testing.T) {
	// 	assert := assert.New(t)
	// 	printer := &TestPrinter{}
	//
	// 	json := []byte(`{"_id":"test1", "v":42}`)
	//
	// 	_, err := cdb.DB().Put(context.TODO(), "test1", json)
	// 	assert.NoError(err)
	//
	// 	editor := &ConflictEditor{cdb: cdb, id: "test1"}
	// 	c := NewTestClippan(cdb,
	// 		true,
	// 		printer,
	// 		editor,
	// 		NewMockPrompt().SetMockData("f"), // f = force
	// 	)
	//
	// 	// Activate the testing database
	// 	c.Executer("use " + cdb.DB().Name())
	// 	c.Executer("edit test1")
	// 	assert.Len(printer.Errors, 0)
	//
	// 	var doc map[string]interface{}
	//
	// 	found, err := helpers.GetOr404(cdb.GetDB(), "test1", &doc)
	// 	assert.NoError(err)
	// 	assert.True(found)
	// 	// The rev should differ from what the editor created!
	// 	assert.NotEqual(editor.revResult, doc["_rev"].(string))
	// 	assert.EqualValues(42, doc["v"].(float64))
	// }))
}

type ConflictEditor struct {
	id        string
	cdb       *helpers.CouchDB
	revResult string
	errResult error
}

func (c *ConflictEditor) Edit(content []byte) ([]byte, error) {
	// fake the conflict by saving it here
	c.revResult, c.errResult = c.cdb.DB().Put(context.TODO(), c.id, content)
	return content, nil
}

func TestQuery(t *testing.T) {
	DB := helpers.DBSession("test-query")

	// Test regular, reduce level 0, reduce level 1
	setUp := func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)
		testview := `{
  "_id": "_design/testview",
  "views": {
    "v1": {
      "map": "function (doc) {\n\tif(doc.type \u0026\u0026 doc.type === \"entry\") {\n\t\t\temit([doc.firstname + \"_\" + doc.lastname], doc.amount);\n\t}\n}",
      "reduce": "_sum"
    }
  }
}`

		_, err := cdb.DB().Put(context.TODO(), "_design/testview", testview)
		assert.NoError(err)

		entry1 := `{
  "_id": "entry1",
  "amount": 123,
  "firstname": "John",
  "lastname": "Doe",
  "type": "entry"
}`
		entry2 := `
{
  "_id": "entry2",
  "amount": 22,
  "firstname": "Jane",
  "lastname": "Doe",
  "type": "entry"
}`
		_, err = cdb.DB().Put(context.TODO(), "entry1", entry1)
		assert.NoError(err)
		_, err = cdb.DB().Put(context.TODO(), "entry2", entry2)
		assert.NoError(err)
	}

	t.Run("Test regular query", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)
		printer := &TestPrinter{}

		setUp(cdb, t)
		json := []byte(`{"_id":"test1", "v":42}`)
		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			NewMockPrompt().SetMockData("a"),
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("query -json testview v1")
		assert.Len(printer.Errors, 0)
		assert.Len(printer.JSONS, 1)

		var res []*QueryResult
		MustUnmarshal(printer.JSONS[0], &res)

		assert.Len(res, 2)
		// Jane comes before John. These assertions make some assumptions about structure and type
		assert.Equal("entry2", res[0].ID)
		assert.EqualValues(22, res[0].Value.(float64))
		assert.EqualValues("Jane_Doe", res[0].Key.([]interface{})[0].(string))
		assert.Equal("entry1", res[1].ID)
		assert.EqualValues(123, res[1].Value.(float64))
		assert.EqualValues("John_Doe", res[1].Key.([]interface{})[0].(string))
	}))
	t.Run("Test reduce query", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)
		printer := &TestPrinter{}

		setUp(cdb, t)
		json := []byte(`{"_id":"test1", "v":42}`)
		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			NewMockPrompt().SetMockData("a"),
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("query -json -reduce testview v1")
		assert.Len(printer.Errors, 0)
		assert.Len(printer.JSONS, 1)

		var res []*QueryResult
		MustUnmarshal(printer.JSONS[0], &res)

		assert.Len(res, 1)
		assert.Equal("", res[0].ID)
		assert.Nil(res[0].Key)
		assert.EqualValues(145, res[0].Value.(float64))
	}))
	t.Run("Test reduce query, level 0", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		// same as not specifying a level at all
		assert := assert.New(t)
		printer := &TestPrinter{}

		setUp(cdb, t)
		json := []byte(`{"_id":"test1", "v":42}`)
		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			NewMockPrompt().SetMockData("a"),
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("query -json -reduce -level=0 testview v1")
		assert.Len(printer.Errors, 0)
		assert.Len(printer.JSONS, 1)

		var res []*QueryResult
		MustUnmarshal(printer.JSONS[0], &res)

		assert.Len(res, 1)
		assert.Equal("", res[0].ID)
		assert.Nil(res[0].Key)
		assert.EqualValues(145, res[0].Value.(float64))
	}))
	t.Run("Test reduce query, level 0", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		// same as not specifying a level at all
		assert := assert.New(t)
		printer := &TestPrinter{}

		setUp(cdb, t)
		json := []byte(`{"_id":"test1", "v":42}`)
		c := NewTestClippan(cdb,
			true,
			printer,
			NewMockEditor().SetMockData(json, nil),
			NewMockPrompt().SetMockData("a"),
		)

		// Activate the testing database
		c.Executer("use " + cdb.DB().Name())
		c.Executer("query -json -reduce -level=1 testview v1")
		assert.Len(printer.Errors, 0)
		assert.Len(printer.JSONS, 1)

		var res []*QueryResult
		MustUnmarshal(printer.JSONS[0], &res)

		// Almost similar to not reducing at all, but no ID's
		assert.Equal("", res[0].ID)
		assert.EqualValues(22, res[0].Value.(float64))
		assert.EqualValues("Jane_Doe", res[0].Key.([]interface{})[0].(string))
		assert.Equal("", res[1].ID)
		assert.EqualValues(123, res[1].Value.(float64))
		assert.EqualValues("John_Doe", res[1].Key.([]interface{})[0].(string))
	}))
}
