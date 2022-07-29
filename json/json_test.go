package json

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {

	t.Run("Test success", func(t *testing.T) {
		sample := []byte("{}")
		var out interface{}
		assert.NoError(t, ValidateUnmarshal(sample, &out))
	})
	t.Run("Test empty", func(t *testing.T) {
		assert := assert.New(t)

		sample := []byte(``)
		var out interface{}
		err := ValidateUnmarshal(sample, &out)
		assert.Error(err)

		ve, match := err.(*ValidationError)
		assert.True(match)
		assert.Equal(0, ve.Line)
		assert.Equal(0, ve.Col)

		// Make helper
		_, match = ve.Err.(*json.SyntaxError)
		assert.True(match)
	})
	t.Run("Test basic failure - missing value", func(t *testing.T) {
		assert := assert.New(t)

		sample := []byte(`{"foo":}`)
		var out interface{}
		err := ValidateUnmarshal(sample, &out)
		assert.Error(err)

		ve, match := err.(*ValidationError)
		assert.True(match)
		assert.Equal(1, ve.Line)
		assert.Equal(8, ve.Col)

		_, match = ve.Err.(*json.SyntaxError)
		assert.True(match)
	})
	t.Run("Test multiline failure - trailing comma", func(t *testing.T) {
		assert := assert.New(t)

		sample := []byte(`{
"foo": 1,
"bar": 2,
}`)
		var out interface{}
		err := ValidateUnmarshal(sample, &out)
		assert.Error(err)

		ve, match := err.(*ValidationError)
		assert.True(match)
		// The comma isn't the parsing issue, the } is unexpected
		assert.Equal(4, ve.Line)
		assert.Equal(1, ve.Col)

		_, match = ve.Err.(*json.SyntaxError)
		assert.True(match)
	})
	t.Run("Test multiline failure - missing closing }", func(t *testing.T) {
		assert := assert.New(t)

		sample := []byte(`{
"foo": 1,
"bar": 2`)
		var out interface{}
		err := ValidateUnmarshal(sample, &out)
		assert.Error(err)

		ve, match := err.(*ValidationError)
		assert.True(match)
		// The parsing will end at the last parsed character
		assert.Equal(3, ve.Line)
		assert.Equal(8, ve.Col)

		_, match = ve.Err.(*json.SyntaxError)
		assert.True(match)
	})
}

func TestAnnotate(t *testing.T) {
	// Just some random json, without actual errors since that's irrelevant
	code := []byte(`{
"results": [
    {
      "aa": 1,
      "bb": 2
    },
    {
      "aa": 3,
      "bb": 4
    },
    {
      "aa": 5,
      "bb": 6
    }
  ]
}`)
	t.Run("In the middle, 1 line context both sides", func(t *testing.T) {
		assert := assert.New(t)
		ve := &ValidationError{
			Line: 5,
			Col:  8,
			Err:  errors.New("Wrong!"),
		}
		res, idx := AnnotateError(code, ve, 1, 1)

		// The line with error, 1 line context before, message after, context after
		assert.Equal(2, idx)
		assert.Len(res, 4)
		assert.Equal(`      "aa": 1,`, res[0])
		assert.Equal(`      "bb": 2`, res[1])
		assert.Equal("       ^-Wrong!", res[2])
		assert.Equal(`    },`, res[3])
	})
	t.Run("In the middle, no context", func(t *testing.T) {
		assert := assert.New(t)
		ve := &ValidationError{
			Line: 5,
			Col:  8,
			Err:  errors.New("Wrong!"),
		}
		res, idx := AnnotateError(code, ve, 0, 0)

		// The line with error, 1 line context before, message after, context after
		assert.Equal(1, idx)
		assert.Len(res, 2)
		assert.Equal(`      "bb": 2`, res[0])
		assert.Equal("       ^-Wrong!", res[1])
	})
	t.Run("In the middle, 2 above 3 below context", func(t *testing.T) {
		assert := assert.New(t)
		ve := &ValidationError{
			Line: 5,
			Col:  13,
			Err:  errors.New("Wrong!"),
		}
		res, idx := AnnotateError(code, ve, 2, 3)

		// The line with error, 1 line context before, message after, context after
		assert.Equal(3, idx)
		assert.Len(res, 7)
		assert.Equal(`    {`, res[0])
		assert.Equal(`      "aa": 1,`, res[1])
		assert.Equal(`      "bb": 2`, res[2])
		assert.Equal("            ^-Wrong!", res[3])
		assert.Equal(`    },`, res[4])
		assert.Equal(`    {`, res[5])
		assert.Equal(`      "aa": 3,`, res[6])
	})
	t.Run("First row", func(t *testing.T) {
		assert := assert.New(t)
		ve := &ValidationError{
			Line: 1,
			Col:  1,
			Err:  errors.New("Wrong!"),
		}
		res, idx := AnnotateError(code, ve, 1, 1)

		// The line with error, 1 line context before, message after, context after
		assert.Equal(1, idx)
		assert.Len(res, 3)
		assert.Equal(`{`, res[0])
		assert.Equal("^-Wrong!", res[1])
		assert.Equal(`"results": [`, res[2])
	})
	t.Run("Last row", func(t *testing.T) {
		assert := assert.New(t)
		ve := &ValidationError{
			Line: 16,
			Col:  1,
			Err:  errors.New("Wrong!"),
		}
		res, idx := AnnotateError(code, ve, 1, 1)

		// The line with error, 1 line context before, message after, context after
		assert.Equal(2, idx)
		assert.Len(res, 3)
		assert.Equal(`  ]`, res[0])
		assert.Equal(`}`, res[1])
		assert.Equal("^-Wrong!", res[2])
	})
}
