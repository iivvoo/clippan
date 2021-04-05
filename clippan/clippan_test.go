package clippan

import (
	"fmt"
	"testing"

	"github.com/iivvoo/clippan/bench"
	"github.com/stretchr/testify/assert"
)

type TestPrinter struct {
	Errors []string // possibly preserve format and args separately
	Debugs []string
	Prints []string
}

func (t *TestPrinter) Error(format string, args ...interface{}) {
	t.Errors = append(t.Errors, fmt.Sprintf(format+"\n", args...))
}
func (t *TestPrinter) Debug(format string, args ...interface{}) {
	t.Debugs = append(t.Debugs, fmt.Sprintf(format+"\n", args...))
}
func (t *TestPrinter) Print(format string, args ...interface{}) {
	t.Prints = append(t.Prints, fmt.Sprintf(format+"\n", args...))
}

func NewTestClippan(testdb *bench.CouchDB, enableWrite bool, printer Printer) *Clippan {
	p := NewPrompt()
	return &Clippan{
		dsn:         "",
		db:          "",
		client:      testdb.Client(),
		enableWrite: enableWrite,
		host:        "",
		prompt:      p,
		Printer:     printer,
	}
}

func TestClippan(t *testing.T) {
	DB := bench.DBSession("test-clippan")

	t.Run("Test UseDB", DB(func(cdb *bench.CouchDB, t *testing.T) {
		assert := assert.New(t)

		c := NewTestClippan(cdb, false, &TestPrinter{})

		// This database does not exist (I hope)
		assert.False(c.UseDB("testKJHANFIU"))
		// This one does exist since testing-<postfix> will be the test database
		assert.True(c.UseDB("testing-test-clippan"))
	}))
}