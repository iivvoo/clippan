package json

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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
	err := json.Unmarshal(data, v)
	if err == nil {
		return nil
	}
	switch err := err.(type) {
	case *json.SyntaxError:
		fmt.Printf("Error in input syntax at byte %d: %s\n", err.Offset, err.Error())
		scanner := bufio.NewScanner(bytes.NewReader(data))
		var line int
		var readBytes int64
		var rowBytes int64
		for scanner.Scan() {
			// +1 for the \n character
			readBytes += int64(len(scanner.Bytes()) + 1)
			line += 1
			if readBytes >= err.Offset {
				fmt.Printf("Error in input syntax on line %d: %s\n", line, err.Error())
				return &ValidationError{Offset: err.Offset, Line: line, Col: int(err.Offset - rowBytes), Err: err}
			}
			rowBytes += readBytes
		}
		// We somehow couldn't find the position, just provide Col as offset on line 0
		return &ValidationError{Offset: err.Offset, Line: 0, Col: int(err.Offset), Err: err}

	default:
		fmt.Println("****", err)
		fmt.Printf("Other error decoding JSON: %s\n", err.Error())
		// line and col 0 are not valid positions, so they can indicate lack of one
		return &ValidationError{Line: 0, Col: 0, Err: err}
	}
	return nil
}

// Given an parse error result, provide textual context: the line with the error (possibly allow annotation)
func AnnotateError() {}
