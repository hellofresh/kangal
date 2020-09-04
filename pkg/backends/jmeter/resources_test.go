package jmeter

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var logger = zap.NewNop()

func TestSplitTestData(t *testing.T) {
	teststring := "aaa \n bbb\n ccc\n"
	testnum := 3

	result, err := splitTestData(teststring, testnum, logger)
	assert.NoError(t, err)
	assert.Equal(t, testnum, len(result))
	assert.Equal(t, "aaa ", string(result[0][0][0]))
}

func TestSplitTestDataEmptyString(t *testing.T) {
	teststring := ""
	testnum := 2

	result, err := splitTestData(teststring, testnum, logger)
	assert.NoError(t, err)
	assert.Equal(t, testnum, len(result))
	for _, r := range result {
		assert.Equal(t, 0, len(r))
	}
}

func TestSplitTestDataEmptyLines(t *testing.T) {
	teststring := "aaa \n \n \n"
	testnum := 2

	result, err := splitTestData(teststring, testnum, logger)
	assert.NoError(t, err)
	assert.Equal(t, "aaa ", string(result[0][0][0]))
	assert.Equal(t, " ", string(result[1][0][0]))
}

func TestSplitTestDataSymbols(t *testing.T) {
	teststring := "onë tw¡™£¢§ˆˆ•ªºœo\n3+4\n dreÄ \nquatr%o\n"
	testnum := 2

	result, err := splitTestData(teststring, testnum, logger)

	assert.NoError(t, err)
	assert.Equal(t, "3+4", string(result[0][1][0]))
	assert.Equal(t, "quatr%o", string(result[1][1][0]))
}

func TestSplitTestDataTrimComma(t *testing.T) {
	teststring := "one, two, tree, four"
	testnum := 1

	result, err := splitTestData(teststring, testnum, logger)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result[0][0]))
	assert.Equal(t, " four", string(result[0][0][3]))
}

func TestReadSecret(t *testing.T) {
	teststring := "aaa,1\nbbb,2\nccc,3\n"

	result, err := readSecret(teststring, logger)
	assert.NoError(t, err)
	assert.Equal(t, int(3), len(result))
	assert.Equal(t, "2", string(result["bbb"]))
}

func TestReadSecretInvalid(t *testing.T) {
	teststring := "aaa:1\nbbb;2\nccc;3\n"
	expectedError := os.ErrInvalid

	_, err := readSecret(teststring, logger)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestReadSecretEmpty(t *testing.T) {
	teststring := ""

	result, err := readSecret(teststring, logger)
	assert.NoError(t, err)
	assert.Equal(t, int(0), len(result))
}

func TestSplitTestDataInvalid(t *testing.T) {
	teststring := "aaa1,rfergerf efesv\nbbb;2\nccc;3\n"
	testnum := 1
	expectedErrorMessage := "record on line 2: wrong number of fields"

	_, err := splitTestData(teststring, testnum, logger)
	assert.Error(t, err)
	assert.Equal(t, expectedErrorMessage, err.Error())
}

func TestGetNamespaceFromName(t *testing.T) {
	teststring := "dummy-name-for-the-test-fake-animal"
	expectedNamespace := "fake-animal"
	res, err := getNamespaceFromLoadTestName(teststring, logger)
	assert.NoError(t, err)
	assert.Equal(t, expectedNamespace, res)
}

func TestGetNamespaceFromInvalidName(t *testing.T) {
	teststring := "dummy_test_fak e_animal"
	expectedError := "invalid argument"
	res, err := getNamespaceFromLoadTestName(teststring, logger)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err.Error())
	assert.Equal(t, "", res)
}
