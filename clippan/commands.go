package clippan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/go-kivik/kivik/v4"
	"github.com/gobwas/glob"
	"github.com/iivvoo/clippan/helpers"
	"github.com/tidwall/pretty"
)

type Flag uint8

const (
	None           Flag = 0
	NeedConnection Flag = iota << 1
	NeedDatabase
)

type Command struct {
	cmd     string
	help    string
	writeOp bool
	flags   Flag
	handler func(*Clippan, []string) error
}

var UsageError = errors.New("Incorrect Usage")
var NoDatabaseError = errors.New("Not connected to a database")
var DocumentNotFoundError = errors.New("Document not found")
var DatabaseExists = errors.New("Database already exists")
var DatabaseDoesNotExist = errors.New("Database does not exist")

var Commands []*Command

func init() {
	Commands = []*Command{
		{"use", "Connect to a database (takes just a database name or a full dsn)", false, NeedConnection, UseDB},
		{"databases", "List all databases", false, NeedConnection, Databases},
		{"createdb", "Create a database", true, NeedConnection, CreateDB},
		{"deletedb", "Delete a database", true, NeedConnection, DeleteDB},
		{"all", "List all docs, paginated", false, NeedDatabase, AllDocs},
		{"get", "Get a single document by id", false, NeedDatabase, Get},
		{"put", "Create a new document", true, NeedDatabase, Put},
		{"edit", "Edit an existing document", true, NeedDatabase, Edit},
		{"query", "Query a view", false, NeedDatabase, Query},
		{"exit", "Exit clippan", false, None, Exit},
		{"help", "Show help", false, None, Help},
	}
}

func MatchDatabases(c *Clippan, patterns ...string) ([]string, []string, error) {
	dbs, err := c.client.AllDBs(context.TODO())
	if err != nil {
		return nil, nil, err
	}
	mismatches := make([]string, 0)
	matches := make([]string, 0)

	for _, pattern := range patterns {
		g := glob.MustCompile(pattern)
		count := 0
		for _, db := range dbs {
			if g.Match(db) {
				matches = append(matches, db)
				count += 1
			}
		}
		if count == 0 {
			mismatches = append(mismatches, pattern)
		}
	}
	return matches, mismatches, nil
}

func CreateDB(c *Clippan, args []string) error {
	// Let's assume we also use it immediately
	if len(args) != 2 {
		return UsageError
	}
	db := args[1]
	if exists, err := c.client.DBExists(context.TODO(), db); err != nil {
		return err
	} else if exists {
		return DatabaseExists
	}
	if err := c.client.CreateDB(context.TODO(), db); err != nil {
		return err
	}
	c.UseDB(db)
	return nil
}

func DeleteDB(c *Clippan, args []string) error {
	// Make sure we disconnect from db if we'e currently connected to it
	force := false

	if len(args) < 2 {
		return UsageError
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.BoolVar(&force, "f", false, "Force operation")
	if fs.Parse(args[1:]) == flag.ErrHelp {
		return nil // help will be printed
	}

	toDelete, mismatches, err := MatchDatabases(c, fs.Args()...)
	if err != nil {
		return err
	}

	for _, db := range toDelete {
		if !force {
			in := c.Prompt.Input("Please type " + db + " to delete it> ")
			if in != db {
				c.Print("Okay, not deleting")
				continue
			}
		}
		if c.database != nil && db == c.database.Name() {
			c.Print("Unselecting database before destroying")
			c.database.Close(context.TODO())
			c.database = nil
			c.db = ""
			c.Prompt.SetPrompt(c.host)
		}
		if err := c.client.DestroyDB(context.TODO(), db); err != nil {
			return err
		}
		c.Print("Database " + db + " destroyed")
	}
	for _, mismatch := range mismatches {
		c.Error("No matches for pattern %s", mismatch)
	}
	return nil
}

func Help(c *Clippan, args []string) error {
	for _, ce := range Commands {
		writeHelp := ""
		if ce.writeOp {
			writeHelp = "(w)"
			if !c.enableWrite {
				writeHelp = "(disabled, ro mode)"
			}
		}
		c.Print("%-20s  %s %s", ce.cmd, ce.help, writeHelp)
	}
	c.Print("\nUse <cmd> -h to get additional options for that command")
	return nil
}

func Databases(c *Clippan, args []string) error {
	long := false

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.BoolVar(&long, "l", false, "Long list format")
	if fs.Parse(args[1:]) == flag.ErrHelp {
		return nil // help will be printed
	}

	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}
	matches, mismatches, err := MatchDatabases(c, patterns...)
	if err != nil {
		return err
	}
	if long {
		stats, err := c.client.DBsStats(context.TODO(), matches)
		if err != nil {
			return err
		}
		c.Print("%-50s %10s %10s", "Name", "#docs", "#deleted")
		for _, s := range stats {
			// Possibly truncate, ellipsize name
			c.Print("%-50s %10d %10d", s.Name, s.DocCount, s.DeletedCount)
		}

	} else {
		for _, db := range matches {
			c.Print(db)
		}
	}
	for _, mismatch := range mismatches {
		c.Error("No matches for pattern %s", mismatch)
	}
	return nil
}

func UseDB(c *Clippan, args []string) error {
	if len(args) != 2 {
		return UsageError
	}
	c.UseDB(args[1])

	return nil
}

// Get returns a single document
func Get(c *Clippan, args []string) error {
	if c.database == nil {
		c.Error("Not connected to a database")
		return NoDatabaseError
	}
	if len(args) != 2 {
		return UsageError
	}
	id := args[1]
	var doc interface{}

	found, err := helpers.GetOr404(c.database, id, &doc)
	if err != nil {
		return err // wrap?
	}
	if !found {
		return DocumentNotFoundError
	}
	// data, err := json.MarshalIndent(doc, "", "  ")
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	c.JSON(data)
	return nil
}

// AllDocs simply returns what _all_docs returns, Will eventually
// support pagination and simple start/end filtering
func AllDocs(c *Clippan, args []string) error {
	if c.database == nil {
		return NoDatabaseError
	}
	var options kivik.Options

	pattern := ""
	if len(args) > 1 {
		pattern = args[1]
	}
	if pattern != "" {
		options = kivik.Options{
			"start_key": pattern,
			"end_key":   pattern + "\ufff0",
		}
	}

	rows, err := c.database.AllDocs(context.TODO(), options)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value interface{}
		if err := rows.ScanKey(&key); err != nil {
			return err
		}
		if err := rows.ScanValue(&value); err != nil {
			return err
		}
		// Abuse json.Marshal to get a representation of value
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		c.Print("%s %v %+v", rows.ID(), key, string(data))
	}
	if rows.Err() != nil {
		return err
	}
	return nil
}

// GetDocRaw gets a document as raw bytes. It returns DocumentNotFoundError
// if not found, or any other error encountered
func GetDocRaw(c *Clippan, id string) ([]byte, map[string]interface{}, error) {
	var doc map[string]interface{}
	found, err := helpers.GetOr404(c.database, id, &doc)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, nil, DocumentNotFoundError
	}

	var data []byte
	if data, err = json.Marshal(&doc); err != nil {
		return nil, nil, err
	}
	return data, doc, nil
}

// Put craetes a new document
func EditPut(c *Clippan, args []string, onlyEdit bool) error {
	if c.database == nil {
		c.Error("Not connected to a database")
		return NoDatabaseError
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)

	if fs.Parse(args[1:]) != nil {
		return nil // help will have been printed
	}
	if fs.NArg() != 1 {
		return UsageError
	}
	id := fs.Arg(0)

	data, _, err := GetDocRaw(c, id)
	// There's no reason not to make it pretty. Fauxton does it as well
	data = pretty.Pretty(data)

	if err != nil && err != DocumentNotFoundError {
		return err
	}
	if err == DocumentNotFoundError {
		data = []byte(`{"_id": "` + id + `"}`)
		if onlyEdit {
			return DocumentNotFoundError
		}
		c.Print("Creating " + id)
	} else {
		if !onlyEdit {
			c.Print("%s already exists, editing in stead", id)
		}
	}

	// as long as we don't successfully safe or get errors
	for {
		data, err = c.Editor.Edit(data)
		if err != nil {
			return err
		}
		if err = ValidateJSON(data); err != nil {
			in := c.Prompt.Input("Document does not validate as json. (E)dit again or (A)bort?> ")
			in = strings.ToLower(in)
			if in == "a" {
				return nil
			}
			continue // try again
		}

		rev, err := c.database.Put(context.TODO(), id, data)
		// Check if conflict, suggest solutions such as
		// - replace
		// - merge-edit
		if err == nil {
			c.Print(rev)
			break
		}

		if kivik.StatusCode(err) == http.StatusConflict {
			newerData, doc, err := GetDocRaw(c, id)
			if err != nil { // even if DocumentNotFoundError because that wouldn't make sense at all
				return err
			}
			rev = doc["_rev"].(string)
			in := c.Prompt.Input("Conflict with rev " + rev + ". (A)bort, [(F)orce] or (E)dit with diff?> ")
			in = strings.ToLower(in)
			if in == "a" {
				return nil
			}
			buf := bytes.NewBuffer(data)
			buf.WriteRune('\n')
			buf.Write(newerData)
			data = buf.Bytes()
			// f would mean replace the rev and save again

			// get data? Or do we get rev? either way we
			// wamt to diff or provide both
		} else {
			// notify user of error so they can perhaps fix issue or retry
			return err
		}
	}
	return nil
}

func ValidateJSON(data []byte) error {
	var x interface{}
	return json.Unmarshal(data, &x)
}

func Edit(c *Clippan, args []string) error {
	return EditPut(c, args, true)
}

func Put(c *Clippan, args []string) error {
	return EditPut(c, args, false)
}

func Exit(c *Clippan, args []string) error {
	c.Print("Bye!")
	os.Exit(0)
	return nil
}

type QueryResult struct {
	ID    string      `json:"id"`
	Key   interface{} `json:"key"`
	Value interface{} `json:"value"`
}

// Query might be aliased / shortcut to Map(view), Reduce?
func Query(c *Clippan, args []string) error {
	/*
	 * Steps:
	 * - query a simple view, list all results
	 */
	var reduce, useJson bool
	var level int

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.BoolVar(&reduce, "reduce", false, "Reduce query")
	fs.IntVar(&level, "level", 0, "Reduce group level")
	fs.BoolVar(&useJson, "json", false, "Output json")
	if fs.Parse(args[1:]) != nil {
		return nil // help will have been printed
	}
	if fs.NArg() != 2 {
		fs.Usage()
		// c.Error("Please specify designdoc and view")
		return nil
	}
	ddoc := fs.Arg(0)
	view := fs.Arg(1)
	c.Print("Querying %s / %s", ddoc, view)

	options := kivik.Options{
		// "startkey":     "",
		// "endkey":       endKey,
		// "include_docs": true,
		"skip":   0,
		"limit":  50,
		"reduce": false,
	}

	if reduce {
		options["reduce"] = true
		options["group_level"] = level
	}
	rows, err := c.database.Query(context.TODO(),
		"_design/"+ddoc, "_view/"+view, options,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var result []*QueryResult

	for rows.Next() {
		var key, value interface{}
		// var doc struct{ _id string }
		if err = rows.ScanKey(&key); err != nil {
			return err
		}
		if err = rows.ScanValue(&value); err != nil {
			return err
		}
		// if err := rows.ScanDoc(&doc); err != nil {
		// 	return err
		// }
		result = append(result, &QueryResult{ID: rows.ID(), Key: key, Value: value})
	}
	if useJson {
		c.JSON(MustMarshal(result))
	} else {
		c.Print("%20v %20v %20s", "Key", "Value", "doc ID")
		for _, r := range result {

			c.Print("%20v %20v %20s", r.Key, r.Value, r.ID)
		}
	}

	return nil
}

func MustMarshal(v interface{}) []byte {
	r, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return r
}

func MustUnmarshal(raw []byte, target interface{}) {
	if err := json.Unmarshal(raw, target); err != nil {
		panic(err)
	}
}
