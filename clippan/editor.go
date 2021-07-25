package clippan

import (
	"io/ioutil"
	"os"
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

	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.Write(content); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}

	return []byte(`{"_id": "frop"}`), nil
}
