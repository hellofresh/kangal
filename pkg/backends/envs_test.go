package backends_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hellofresh/kangal/pkg/backends"
)

func TestReadSecret(t *testing.T) {
	teststring := "aaa,1\nbbb,2\nccc,3\n"

	result, err := backends.ReadEnvs(teststring)
	assert.NoError(t, err)
	assert.Equal(t, int(3), len(result))
	assert.Equal(t, "2", string(result["bbb"]))
}

func TestReadSecretInvalid(t *testing.T) {
	teststring := "aaa:1\nbbb;2\nccc;3\n"
	expectedError := backends.ErrInvalidCSVFormat

	_, err := backends.ReadEnvs(teststring)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestReadSecretEmpty(t *testing.T) {
	teststring := ""

	result, err := backends.ReadEnvs(teststring)
	assert.NoError(t, err)
	assert.Equal(t, int(0), len(result))
}
