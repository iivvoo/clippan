package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {

	t.Run("Test success", func(t *testing.T) {
		json := []byte("{}")
		var out interface{}
		assert.NoError(t, ValidateUnmarshal(json, &out))
	})
	t.Run("Test failure", func(t *testing.T) {
		assert := assert.New(t)

		json := []byte(`{"foo":}`)
		var out interface{}
		err := ValidateUnmarshal(json, &out)
		assert.Error(err)

		ve, match := err.(*ValidationError)
		assert.True(match)
		assert.Equal(1, ve.Line)
		assert.Equal(8, ve.Col)
	})
}
