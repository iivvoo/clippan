package json

import (
	"encoding/json"
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
