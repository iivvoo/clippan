package clippan

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"

	"github.com/c-bata/go-prompt"
	"github.com/go-kivik/kivik/v4"
	"github.com/gobwas/glob"
	"github.com/iivvoo/clippan/bench"
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
		{"databases", "List all databases", false, NeedConnection, Databases},
		{"use", "Connect to a database", false, NeedConnection, UseDB},
		{"createdb", "Create a database", true, NeedConnection, CreateDB},
		{"deletedb", "Delete a database", true, NeedConnection, DeleteDB},
		{"all", "List all docs, paginated", false, NeedDatabase, AllDocs},
		{"get", "Get a single document by id", false, NeedDatabase, Get},
		{"exit", "Exit clippan", false, None, Exit},
		{"help", "Show help", false, None, Help},
	}
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
	fs.Parse(args[1:])

	// We can get multiple args, and some might be wildcards.
	// Expand them to a full list. What if some don't match?
	// `rm` will just delete whatever possible and complain about the rest

	pattern := fs.Arg(0)
	dbs, err := c.client.AllDBs(context.TODO())
	if err != nil {
		return err
	}
	toDelete := make([]string, 0)

	g := glob.MustCompile(pattern)
	matches := 0
	for _, db := range dbs {
		if g.Match(db) {
			toDelete = append(toDelete, db)
			matches += 1
		} else {
			// c.Print("Does not exist " + db)
		}
	}
	if matches == 0 {
		c.Error("%s does not match any database", pattern)
	}

	// if exists, err := c.client.DBExists(context.TODO(), db); err != nil {
	// 	return err
	// } else if !exists {
	// 	return DatabaseDoesNotExist
	// }

	for _, db := range toDelete {
		if !force {
			in := prompt.Input("Please type "+db+" to delete it> ", func(prompt.Document) []prompt.Suggest {
				return nil
			})
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
			c.prompt.SetPrompt(c.host)
		}
		if err := c.client.DestroyDB(context.TODO(), db); err != nil {
			return err
		}
		c.Print("Database " + db + " destroyed")
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
	return nil
}

func Databases(c *Clippan, args []string) error {
	dbs, err := c.client.AllDBs(context.TODO())
	if err != nil {
		return err
	}
	for _, db := range dbs {
		c.Print(db)
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

/*
 * Also, c.Error() in stead of all the Println's
 */

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

	found, err := bench.GetOr404(c.database, id, &doc)
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

func Exit(c *Clippan, args []string) error {
	c.Print("Bye!")
	os.Exit(0)
	return nil
}
