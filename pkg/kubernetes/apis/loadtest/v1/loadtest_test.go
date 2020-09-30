package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHash(t *testing.T) {
	assert.Equal(t, "da39a3ee5e6b4b0d3255bfef95601890afd80709", getHashFromString(""))
}

func TestBuildLoadTestObject(t *testing.T) {
	ltType := LoadTestTypeJMeter
	expectedDP := int32(2)

	spec := LoadTestSpec{
		Type:            ltType,
		DistributedPods: &expectedDP,
		TestFile:        "load-test file\n",
		TestData:        "test data 1\ntest data 2\n",
		EnvVars:         "envVar1,value1\nenvVar2,value2\n",
	}

	expectedLt := LoadTest{
		TypeMeta:   metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{},
		Spec:       spec,
		Status: LoadTestStatus{
			Phase: LoadTestCreating,
		},
	}

	lt, err := BuildLoadTestObject(spec)
	assert.NoError(t, err)
	assert.Equal(t, expectedLt.Spec, lt.Spec)
	assert.Equal(t, expectedLt.Status.Phase, lt.Status.Phase)
}
