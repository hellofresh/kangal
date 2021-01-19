package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadSecret(t *testing.T) {
	teststring := "aaa,1\nbbb,2\nccc,3\n"

	result, err := ReadEnvs(teststring)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, "2", result["bbb"])
}

func TestReadSecretInvalid(t *testing.T) {
	teststring := "aaa:1\nbbb;2\nccc;3\n"
	expectedError := ErrInvalidCSVFormat

	_, err := ReadEnvs(teststring)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestReadSecretEmpty(t *testing.T) {
	teststring := ""

	result, err := ReadEnvs(teststring)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result))
}
