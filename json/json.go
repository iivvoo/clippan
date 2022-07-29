package json

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Validate json, return line, column of any issue + type of error
// (can we differentiate on err)

type ValidationError struct {
	Line   int
	Col    int
	Offset int64
	Err    error
}

func (v *ValidationError) Error() string {
	return fmt.Sprintf("Syntax error at row %d col %d: %v", v.Line, v.Col, v.Err)
}

func ValidateUnmarshal(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err == nil {
		return nil
	} else if err, m := err.(*json.SyntaxError); m {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		var line int
		var bytesRead int64
		var rowOffset int64
		for scanner.Scan() {
			// +1 for the \n character
			rowOffset = bytesRead
			bytesRead += int64(len(scanner.Bytes()) + 1)
			line += 1
			if bytesRead >= err.Offset {
				return &ValidationError{Offset: err.Offset, Line: line, Col: int(err.Offset - rowOffset), Err: err}
			}
		}
		// We somehow couldn't find the position, just provide Col as offset on line 0
		return &ValidationError{Offset: err.Offset, Line: 0, Col: int(err.Offset), Err: err}
	} else {
		// line and col 0 are not valid positions, so they can indicate lack of one
		return &ValidationError{Line: 0, Col: 0, Err: err}
	}
	return nil
}

// Given the ValidationError, produce an error message with context, as a slice of strings
// the returned int will indicated the index of the error itself
func AnnotateError(data []byte, issue *ValidationError, before, after int) ([]string, int) {
	context := make([]string, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	index := 0
	messagePos := 0

	for scanner.Scan() {
		index += 1
		if index >= issue.Line-before && index <= issue.Line+after {
			line := string(scanner.Bytes())
			context = append(context, line)

			if index == issue.Line {
				message := strings.Repeat(" ", issue.Col-1) + "^-" + issue.Err.Error()
				context = append(context, message)
				messagePos = len(context) - 1
			}
		}
	}
	return context, messagePos
}
