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

// Actually edit a file using a real editor and a tempfile
type RealEditor struct {
	editor string
}

func NewRealEditor(editor string) *RealEditor {
	return &RealEditor{editor}
}

func (r *RealEditor) Edit(content []byte) ([]byte, error) {
	tmpfile, err := ioutil.TempFile("", "clippan")
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
	cmd := exec.Command(r.editor, tmpfile.Name())
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
