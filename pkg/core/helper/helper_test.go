package helper_test

import (
	"testing"

	"github.com/hellofresh/kangal/pkg/core/helper"
	"github.com/stretchr/testify/assert"
)

func TestReadSecret(t *testing.T) {
	teststring := "aaa,1\nbbb,2\nccc,3\n"

	result, err := helper.ReadEnvs(teststring)
	assert.NoError(t, err)
	assert.Equal(t, int(3), len(result))
	assert.Equal(t, "2", string(result["bbb"]))
}

func TestReadSecretInvalid(t *testing.T) {
	teststring := "aaa:1\nbbb;2\nccc;3\n"
	expectedError := helper.ErrInvalidCSVFormat

	_, err := helper.ReadEnvs(teststring)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestReadSecretEmpty(t *testing.T) {
	teststring := ""

	result, err := helper.ReadEnvs(teststring)
	assert.NoError(t, err)
	assert.Equal(t, int(0), len(result))
}
