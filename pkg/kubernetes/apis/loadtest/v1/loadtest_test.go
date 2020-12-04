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
		Tags:            map[string]string{"department": "platform", "team": "kangal"},
		TestFile:        "load-test file\n",
		TestData:        "test data 1\ntest data 2\n",
		EnvVars:         "envVar1,value1\nenvVar2,value2\n",
	}

	expectedLabels := map[string]string{
		"test-file-hash":      "5a7919885ef46f2e0bd66602944128fde2dce928",
		"test-tag-department": "platform",
		"test-tag-team":       "kangal",
	}

	expectedLt := LoadTest{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Labels: expectedLabels,
		},
		Spec: spec,
		Status: LoadTestStatus{
			Phase: LoadTestCreating,
		},
	}

	lt, err := BuildLoadTestObject(spec)
	assert.NoError(t, err)
	assert.Equal(t, expectedLt.ObjectMeta.Labels, lt.ObjectMeta.Labels)
	assert.Equal(t, expectedLt.Spec, lt.Spec)
	assert.Equal(t, expectedLt.Status.Phase, lt.Status.Phase)
}

func TestLoadTestTagsFromString(t *testing.T) {
	testCases := []struct {
		scenario       string
		input          string
		expectedResult LoadTestTags
		expectedError  string
	}{
		{
			scenario:       "no input no error",
			expectedResult: LoadTestTags{},
		},
		{
			scenario:       "no value no error",
			input:          ",",
			expectedResult: LoadTestTags{},
		},
		{
			scenario:      "missing label",
			input:         ":value-only",
			expectedError: "missing tag label",
		},
		{
			scenario:      "missing value",
			input:         "label:",
			expectedError: "missing tag value",
		},
		{
			scenario:      "value is too long",
			input:         "label:MW5Ex91GtG5qTRnC2DIxWo17t6yjkJBCtp9Mh5q0J7R7RXDcoAvRcYmL5Uqc8YeR",
			expectedError: "tag value is too long",
		},
		{
			scenario: "multiple tags",
			input:    "tag1:value1,tag2:value2,,,,tag3:value3",
			expectedResult: LoadTestTags{
				"tag1": "value1",
				"tag2": "value2",
				"tag3": "value3",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			result, err := LoadTestTagsFromString(tc.input)

			assert.Equal(t, tc.expectedResult, result)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestLoadTestPhaseFromString(t *testing.T) {
	for _, tt := range []struct {
		name string
		in   string
		out  LoadTestPhase
		err  error
	}{
		{
			name: "empty",
			in:   "",
			out:  "",
			err:  nil,
		},
		{
			name: "creating",
			in:   "creating",
			out:  LoadTestCreating,
			err:  nil,
		},
		{
			name: "random case creating",
			in:   "CreatING",
			out:  LoadTestCreating,
			err:  nil,
		},
		{
			name: "starting",
			in:   "starting",
			out:  LoadTestStarting,
			err:  nil,
		},
		{
			name: "running",
			in:   "running",
			out:  LoadTestRunning,
			err:  nil,
		},
		{
			name: "finished",
			in:   "finished",
			out:  LoadTestFinished,
			err:  nil,
		},
		{
			name: "errored",
			in:   "errored",
			out:  LoadTestErrored,
			err:  nil,
		},
		{
			name: "invalid",
			in:   "foobar",
			out:  "",
			err:  ErrUnknownLoadTestPhase,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := LoadTestPhaseFromString(tt.in)
			assert.Equal(t, tt.out, out)
			assert.Equal(t, tt.err, err)
		})
	}
}
