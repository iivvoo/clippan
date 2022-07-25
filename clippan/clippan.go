package clippan

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	_ "github.com/go-kivik/couchdb/v4"
	"github.com/go-kivik/kivik/v4"
	"github.com/mattn/go-shellwords"
	"github.com/tidwall/pretty"
)

type Printer interface {
	Error(string, ...interface{})
	Debug(string, ...interface{})
	Print(string, ...interface{})
	JSON([]byte)
}
type TextPrinter struct {
	debug bool
}

func (p *TextPrinter) Error(format string, args ...interface{}) {
	fmt.Printf("ERROR: "+format+"\n", args...)
}

func (p *TextPrinter) Debug(format string, args ...interface{}) {
	if p.debug {
		fmt.Printf("DEBUG: "+format+"\n", args...)
	}
}

func (p *TextPrinter) Print(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func (p *TextPrinter) JSON(raw []byte) {
	data := pretty.Pretty(raw)
	// can be optional if we want color / more control over formatting
	data = pretty.Color(data, nil)
	fmt.Println(string(data))
}

type Clippan struct {
	dsn      string
	client   *kivik.Client
	database *kivik.DB

	// DI
	Printer Printer
	Editor  Editor
	Prompt  Prompter
	//
	enableWrite bool
	host        string
	db          string // database.Name() ??
}

func NewClippan(dsn string, enableWrite, debug bool) *Clippan {
	u, err := url.Parse(dsn)
	if err != nil {
		panic(err)
	}
	database := strings.Trim(u.Path, "/")
	u.Path = ""
	dsn = u.String()

	editBin := os.Getenv("EDITOR")
	if editBin == "" {
		editBin = "/usr/bin/vi"
	}
	editor := NewRealEditor(editBin)
	// p := NewPrompt()
	return &Clippan{
		dsn:         dsn,
		db:          database,
		client:      nil,
		enableWrite: enableWrite,
		host:        u.Host,
		Prompt:      nil,
		Printer:     &TextPrinter{debug},
		Editor:      editor,
	}
}

func (c *Clippan) Error(format string, args ...interface{}) {
	c.Printer.Error(format, args...)
}

func (c *Clippan) Debug(format string, args ...interface{}) {
	c.Printer.Debug(format, args...)
}

func (c *Clippan) Print(format string, args ...interface{}) {
	c.Printer.Print(format, args...)
}

func (c *Clippan) JSON(raw []byte) {
	c.Printer.JSON(raw)
}

func (c *Clippan) Connect() error {
	if c.client != nil {
		c.client.Close(context.TODO())
	}
	client, err := kivik.New("couch", c.dsn)
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

func (c *Clippan) Executer(s string) {
	parsed, err := shellwords.Parse(s)
	if err != nil {
		c.Error(err.Error())
		return
	}
	if len(parsed) == 0 {
		return
	}
	c.Debug("Command: %#v", parsed)
	cmd := parsed[0]

	found := false
	for _, ce := range Commands {
		if ce.cmd == cmd {
			if ce.writeOp && !c.enableWrite {
				c.Error("Write operation in ro mode. Restart with `-write`")
			} else if ce.flags&NeedConnection == NeedConnection && c.client == nil {
				c.Error("Not connected")
			} else if ce.flags&NeedDatabase == NeedDatabase && c.database == nil {
				c.Error("No database selected")
			} else if err := ce.handler(c, parsed); err != nil {
				c.Error(err.Error())
			}
			found = true
		}
	}
	if !found {
		c.Error("command not found. Use 'help'")
	}
}

func (c *Clippan) UseDB(db string) bool {
	if exists, err := c.client.DBExists(context.TODO(), db); err != nil {
		c.Error(err.Error())
		return false
	} else if !exists {
		c.Error(db + " does not exist")
		return false
	}

	if c.database != nil {
		c.database.Close(context.TODO())
	}

	c.db = db
	c.database = c.client.DB(context.TODO(), db)
	mode := "(ro)"
	if c.enableWrite {
		mode = "(rw)"
	}
	c.Prompt.SetPrompt(c.host + "/" + c.db + mode)
	return true
}

// splitCmds splits ;-separated command line supplied commands into individual commands
func (c *Clippan) splitCmds(cmds string) []string {
	split := []string{}
	if cmds != "" {
		for _, cmd := range strings.Split(cmds, ";") {
			if strings.TrimSpace(cmd) != "" {
				split = append(split, cmd)
			}
		}
	}
	return split
}

// RunCmds runs all individual commands in `cmds`
func (c *Clippan) RunCmds(cmds []string) {
	for _, cmd := range cmds {
		c.Executer(cmd)
	}
}

// Run starts clippan, connecting to the provided dsn (may be empty, may contain database). The optional (can be empty) cmd is a set of commands to parse
func (c *Clippan) Run(cmds string) {
	// for now let's assume it's without a database. We'll need to split anyway
	fullDSN := c.dsn
	if c.db != "" {
		fullDSN += "/" + c.db
	}
	c.Print("Connecting to " + fullDSN)

	split := c.splitCmds(cmds)
	// Initialize prompt with history, if possible
	if c.Prompt == nil {
		c.Prompt = NewPrompt(split...)
	}

	if err := c.Connect(); err != nil {
		c.Error(err.Error())
	} else if c.db != "" {
		c.UseDB(c.db)
	} else {
		c.Prompt.SetPrompt(c.host)
	}

	// if c.database then db = c.client.DB(context.TODO(), c.database)
	c.RunCmds(split)
	c.Prompt.GetInput(c.Executer)
}
