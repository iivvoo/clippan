package clippan

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

type Editor interface {
	Edit([]byte) ([]byte, error)
}

type RealEditor struct{}

func (r *RealEditor) Edit(content []byte) ([]byte, error) {
	// Create a temp file with "existing"
	// find suitable editor, based on $EDITOR and other settings
	// spawn editor, wait
	// re-read temp file, return that data
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		return nil, err
	}

	// defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.Write(content); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}
	// get $EDITOR, from env or options
	// (options should be initialized with env $EDITOR, if not set)
	fmt.Println("Spawning editor on " + tmpfile.Name())
	cmd := exec.Command("/usr/bin/nvim", tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		return nil, err
	}

	dat, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		return nil, err
	}

	return dat, nil
}
