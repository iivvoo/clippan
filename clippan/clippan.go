package clippan

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/go-kivik/couchdb/v4"
	"github.com/go-kivik/kivik/v4"
	"github.com/mattn/go-shellwords"
)

type Clippan struct {
	dsn      string
	client   *kivik.Client
	database *kivik.DB
	//
	enableWrite bool
	host        string
	db          string // database.Name() ??
	prompt      *Prompt
}

func NewClippan(dsn string, enableWrite bool) *Clippan {
	u, err := url.Parse(dsn)
	if err != nil {
		panic(err)
	}
	database := strings.Trim(u.Path, "/")
	u.Path = ""
	dsn = u.String()

	p := NewPrompt()
	return &Clippan{
		dsn:         dsn,
		db:          database,
		client:      nil,
		enableWrite: enableWrite,
		host:        u.Host,
		prompt:      p,
	}
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
		fmt.Println("ERROR: " + err.Error())
		return
	}
	if len(parsed) == 0 {
		return
	}
	fmt.Printf("DEBUG: Command: %#v\n", parsed)
	cmd := parsed[0]

	found := false
	for _, ce := range Commands {
		if ce.cmd == cmd {
			if ce.writeOp && !c.enableWrite {
				fmt.Println("ERROR: Write operation in ro mode. Restart with `-write`")
			} else if ce.flags&NeedConnection == NeedConnection && c.client == nil {
				fmt.Println("ERROR: Not connected")
			} else if ce.flags&NeedDatabase == NeedDatabase && c.database == nil {
				fmt.Println("ERROR: No database selected")
			} else if err := ce.handler(c, parsed); err != nil {
				fmt.Println("ERROR: " + err.Error())
			}
			found = true
		}
	}
	if !found {
		fmt.Println("ERROR: command not found. Use 'help'")
	}
}

func (c *Clippan) UseDB(db string) {
	if exists, err := c.client.DBExists(context.TODO(), db); err != nil {
		fmt.Println("ERROR: " + err.Error())
		return
	} else if !exists {
		fmt.Println("ERROR: " + db + " does not exist")
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
	c.prompt.SetPrompt(c.host + "/" + c.db + mode)

}

// Run starts clippan, connecting to the provided dsn (may be empty, may contain database)
func (c *Clippan) Run() {
	// for now let's assume it's without a database. We'll need to split anyway
	fullDSN := c.dsn
	if c.db != "" {
		fullDSN += "/" + c.db
	}
	fmt.Println("Connecting to " + fullDSN)

	if err := c.Connect(); err != nil {
		fmt.Println("ERROR: " + err.Error())
	} else if c.db != "" {
		c.UseDB(c.db)
	} else {
		c.prompt.SetPrompt(c.host)
	}

	// if c.database then db = c.client.DB(context.TODO(), c.database)
	c.prompt.GetInput(c.Executer)
}
