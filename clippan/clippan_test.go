package clippan

import (
	"fmt"
	"testing"

	"github.com/iivvoo/clippan/helpers"
	"github.com/stretchr/testify/assert"
)

type TestPrinter struct {
	Errors []string // possibly preserve format and args separately
	Debugs []string
	Prints []string
	JSONS  [][]byte
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
func (t *TestPrinter) JSON(raw []byte) {
	t.JSONS = append(t.JSONS, raw)
}

type MockEditor struct {
	result []byte
	err    error
}

func NewMockEditor() *MockEditor {
	return &MockEditor{}
}
func (m *MockEditor) SetMockData(result []byte, err error) *MockEditor {
	m.result = result
	m.err = err
	return m
}

func (m *MockEditor) Edit(content []byte) ([]byte, error) {
	return m.result, m.err
}

type MockPrompt struct {
	result string
	Inputs []string
}

func NewMockPrompt() *MockPrompt {
	return &MockPrompt{}
}
func (m *MockPrompt) SetMockData(result string) *MockPrompt {
	m.result = result
	return m
}
func (m *MockPrompt) GetInput(func(string)) {}
func (m *MockPrompt) SetPrompt(string)      {}
func (m *MockPrompt) Input(s string) string {
	m.Inputs = append(m.Inputs, s)
	return m.result
}
func NewTestClippan(testdb *helpers.CouchDB, enableWrite bool, printer Printer, editor Editor, prompt Prompter) *Clippan {
	// p := NewPrompt()
	return &Clippan{
		dsn:         "",
		db:          "",
		client:      testdb.Client(),
		enableWrite: enableWrite,
		host:        "",
		Prompt:      prompt,
		Printer:     printer,
		Editor:      editor,
	}
}

func TestClippan(t *testing.T) {
	DB := helpers.DBSession("test-clippan")

	t.Run("Test UseDB", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)

		c := NewTestClippan(cdb, false, &TestPrinter{}, NewMockEditor(), NewMockPrompt())

		// This database does not exist (I hope)
		assert.False(c.UseDB("testKJHANFIU"))
		// This one does exist since testing-<postfix> will be the test database
		assert.True(c.UseDB("testing-test-clippan"))
	}))
}

func TestRun(t *testing.T) {
	DB := helpers.DBSession("test-clippan-run")

	t.Run("Test Run with cmd", DB(func(cdb *helpers.CouchDB, t *testing.T) {
		assert := assert.New(t)
		p := &TestPrinter{}
		c := NewTestClippan(cdb, false, p, NewMockEditor(), NewMockPrompt())
		c.RunCmds("a;b -c")

		assert.Len(p.Errors, 2)
		assert.Len(p.Debugs, 2)
		assert.Equal("xCommand: []string{\"a\"}\n", p.Debugs[0])
		assert.Equal("Command: []string{\"b\", \"-c\"}\n", p.Debugs[1])
	}))

}
