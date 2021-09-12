package clippan

import (
	"github.com/c-bata/go-prompt"
)

type Prompt struct {
	prompt    *prompt.Prompt
	ps        string
	_executer func(string)
}

func NewPrompt() *Prompt {
	p := &Prompt{
		prompt:    nil,
		ps:        "",
		_executer: nil,
	}
	p.prompt = prompt.New(p.executer, p.completer,
		prompt.OptionPrefix(">"),
		prompt.OptionLivePrefix(p.livePrefix),
	)
	return p
}

func (p *Prompt) livePrefix() (string, bool) {
	return p.ps + "> ", true
}

/*
 * Basic Prompt / IO implementation. Not sure which "readline-like" package to use,
 * so let's make it somewhat pluggable
 */

func (p *Prompt) completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		// {Text: "users", Description: "Store the username and age"},
		// {Text: "articles", Description: "Store the article text posted by user"},
		// {Text: "comments", Description: "Store the text commented to articles"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func (p *Prompt) executer(s string) {
	if p._executer != nil {
		p._executer(s)
	}
}

func (p *Prompt) SetPrompt(s string) {
	p.ps = s
}

// GetInput gets input from the user and calls the provided callback
func (p *Prompt) GetInput(executer func(string)) {
	p._executer = executer
	p.prompt.Run()
}

// Request simple input
func (p *Prompt) Input(s string) string {
	return prompt.Input(s,
		func(prompt.Document) []prompt.Suggest {
			return nil
		},
	)
}

type Prompter interface {
	GetInput(func(string))
	SetPrompt(string)
	Input(string) string
}
