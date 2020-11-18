package jmeter

import (
	"testing"

	"github.com/hellofresh/kangal/pkg/core/helper"
	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
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

func TestPodResourceConfiguration(t *testing.T) {
	loadTest := loadtestV1.LoadTest{
		Spec: loadtestV1.LoadTestSpec{
			MasterConfig: loadtestV1.ImageDetails{
				Image: defaultMasterImageName,
				Tag:   defaultMasterImageTag,
			},
			WorkerConfig: loadtestV1.ImageDetails{
				Image: defaultWorkerImageName,
				Tag:   defaultWorkerImageTag,
			},
		},
	}

	c := &JMeter{
		masterResources: helper.Resources{
			CPULimits:      "100m",
			CPURequests:    "200m",
			MemoryLimits:   "100Mi",
			MemoryRequests: "200Mi",
		},
		workerResources: helper.Resources{
			CPULimits:      "300m",
			CPURequests:    "400m",
			MemoryLimits:   "300Mi",
			MemoryRequests: "400Mi",
		},
	}

	masterJob := c.NewJMeterMasterJob(loadTest, "http://kangal-proxy.local/load-test/loadtest-name/report", map[string]string{"": ""})
	assert.Equal(t, c.masterResources.CPULimits, masterJob.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
	assert.Equal(t, c.masterResources.CPURequests, masterJob.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
	assert.Equal(t, c.masterResources.MemoryLimits, masterJob.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	assert.Equal(t, c.masterResources.MemoryRequests, masterJob.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())

	workerPod := c.NewPod(loadTest, 0, &v1.ConfigMap{}, map[string]string{"": ""})
	assert.Equal(t, c.workerResources.CPULimits, workerPod.Spec.Containers[0].Resources.Limits.Cpu().String())
	assert.Equal(t, c.workerResources.CPURequests, workerPod.Spec.Containers[0].Resources.Requests.Cpu().String())
	assert.Equal(t, c.workerResources.MemoryLimits, workerPod.Spec.Containers[0].Resources.Limits.Memory().String())
	assert.Equal(t, c.workerResources.MemoryRequests, workerPod.Spec.Containers[0].Resources.Requests.Memory().String())
}
