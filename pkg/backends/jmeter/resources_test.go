package jmeter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	coreV1 "k8s.io/api/core/v1"

	"github.com/hellofresh/kangal/pkg/backends"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestSplitTestData(t *testing.T) {
	teststring := "aaa \n bbb\n ccc\n"
	testnum := 3

	logger := zaptest.NewLogger(t)

	result, err := splitTestData(teststring, testnum, logger)
	assert.NoError(t, err)
	assert.Equal(t, testnum, len(result))
	assert.Equal(t, "aaa ", result[0][0][0])
}

func TestSplitTestDataEmptyString(t *testing.T) {
	teststring := ""
	testnum := 2

	logger := zaptest.NewLogger(t)

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

	logger := zaptest.NewLogger(t)

	result, err := splitTestData(teststring, testnum, logger)
	assert.NoError(t, err)
	assert.Equal(t, "aaa ", result[0][0][0])
	assert.Equal(t, " ", result[1][0][0])
}

func TestSplitTestDataSymbols(t *testing.T) {
	teststring := "onë tw¡™£¢§ˆˆ•ªºœo\n3+4\n dreÄ \nquatr%o\n"
	testnum := 2

	logger := zaptest.NewLogger(t)

	result, err := splitTestData(teststring, testnum, logger)

	assert.NoError(t, err)
	assert.Equal(t, "3+4", result[0][1][0])
	assert.Equal(t, "quatr%o", result[1][1][0])
}

func TestSplitTestDataTrimComma(t *testing.T) {
	teststring := "one, two, tree, four"
	testnum := 1

	logger := zaptest.NewLogger(t)

	result, err := splitTestData(teststring, testnum, logger)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result[0][0]))
	assert.Equal(t, " four", result[0][0][3])
}

func TestSplitTestDataInvalid(t *testing.T) {
	teststring := "aaa1,rfergerf efesv\nbbb;2\nccc;3\n"
	testnum := 1
	expectedErrorMessage := "record on line 2: wrong number of fields"

	logger := zaptest.NewLogger(t)

	_, err := splitTestData(teststring, testnum, logger)
	assert.Error(t, err)
	assert.Equal(t, expectedErrorMessage, err.Error())
}

func TestGetNamespaceFromName(t *testing.T) {
	teststring := "dummy-name-for-the-test-fake-animal"
	expectedNamespace := "fake-animal"
	logger := zaptest.NewLogger(t)
	res, err := getNamespaceFromLoadTestName(teststring, logger)
	assert.NoError(t, err)
	assert.Equal(t, expectedNamespace, res)
}

func TestGetNamespaceFromInvalidName(t *testing.T) {
	teststring := "dummy_test_fak e_animal"
	expectedError := "invalid argument"
	logger := zaptest.NewLogger(t)
	res, err := getNamespaceFromLoadTestName(teststring, logger)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err.Error())
	assert.Equal(t, "", res)
}

func TestPodResourceConfiguration(t *testing.T) {
	lt := loadTestV1.LoadTest{
		Spec: loadTestV1.LoadTestSpec{
			MasterConfig: loadTestV1.ImageDetails(fmt.Sprintf("%s:%s", defaultMasterImageName, defaultMasterImageTag)),
			WorkerConfig: loadTestV1.ImageDetails(fmt.Sprintf("%s:%s", defaultWorkerImageName, defaultWorkerImageTag)),
		},
	}

	c := &Backend{
		logger: zaptest.NewLogger(t),
		masterResources: backends.Resources{
			CPULimits:      "100m",
			CPURequests:    "200m",
			MemoryLimits:   "100Mi",
			MemoryRequests: "200Mi",
		},
		workerResources: backends.Resources{
			CPULimits:      "300m",
			CPURequests:    "400m",
			MemoryLimits:   "300Mi",
			MemoryRequests: "400Mi",
		},
	}

	masterJob := c.NewJMeterMasterJob(lt, "http://kangal-proxy.local/load-test/loadtest-name/report", map[string]string{"": ""})
	assert.Equal(t, c.masterResources.CPULimits, masterJob.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	assert.Equal(t, c.masterResources.CPURequests, masterJob.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
	assert.Equal(t, c.masterResources.MemoryLimits, masterJob.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	assert.Equal(t, c.masterResources.MemoryRequests, masterJob.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())

	workerPod := c.NewPod(lt, 0, &coreV1.ConfigMap{}, map[string]string{"": ""})
	assert.Equal(t, c.workerResources.CPULimits, workerPod.Spec.Containers[0].Resources.Limits.Cpu().String())
	assert.Equal(t, c.workerResources.CPURequests, workerPod.Spec.Containers[0].Resources.Requests.Cpu().String())
	assert.Equal(t, c.workerResources.MemoryLimits, workerPod.Spec.Containers[0].Resources.Limits.Memory().String())
	assert.Equal(t, c.workerResources.MemoryRequests, workerPod.Spec.Containers[0].Resources.Requests.Memory().String())
}
