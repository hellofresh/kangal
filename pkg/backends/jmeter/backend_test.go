package jmeter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestSetDefaults(t *testing.T) {
	t.Run("With env default", func(t *testing.T) {
		jmeter := &Backend{
			config: &Config{
				MasterImageName: "my-master-image-name",
				MasterImageTag:  "my-master-image-tag",
				WorkerImageName: "my-worker-image-name",
				WorkerImageTag:  "my-worker-image-tag",
			},
		}
		jmeter.SetDefaults()

		assert.Equal(t, jmeter.workerConfig.Image, "my-worker-image-name")
		assert.Equal(t, jmeter.workerConfig.Tag, "my-worker-image-tag")
		assert.Equal(t, jmeter.masterConfig.Image, "my-master-image-name")
		assert.Equal(t, jmeter.masterConfig.Tag, "my-master-image-tag")
	})

	t.Run("No default", func(t *testing.T) {
		jmeter := &Backend{
			config: &Config{},
		}
		jmeter.SetDefaults()

		assert.Equal(t, jmeter.masterConfig.Image, defaultMasterImageName)
		assert.Equal(t, jmeter.masterConfig.Tag, defaultMasterImageTag)
		assert.Equal(t, jmeter.workerConfig.Image, defaultWorkerImageName)
		assert.Equal(t, jmeter.workerConfig.Tag, defaultWorkerImageTag)
	})
}

func TestTransformLoadTestSpec(t *testing.T) {
	jmeter := &Backend{
		masterConfig: loadTestV1.ImageDetails{
			Image: "master-image",
			Tag:   "master-tag",
		},
		workerConfig: loadTestV1.ImageDetails{
			Image: "worker-image",
			Tag:   "worker-tag",
		},
	}

	spec := &loadTestV1.LoadTestSpec{}

	t.Run("Empty spec", func(t *testing.T) {
		err := jmeter.TransformLoadTestSpec(spec)
		assert.EqualError(t, err, ErrRequireMinOneDistributedPod.Error())
	})

	t.Run("Negative distributedPods", func(t *testing.T) {
		distributedPods := int32(-10)
		spec.DistributedPods = &distributedPods
		err := jmeter.TransformLoadTestSpec(spec)
		assert.EqualError(t, err, ErrRequireMinOneDistributedPod.Error())
	})

	t.Run("Empty testFile", func(t *testing.T) {
		distributedPods := int32(2)
		spec.DistributedPods = &distributedPods
		err := jmeter.TransformLoadTestSpec(spec)
		assert.EqualError(t, err, ErrRequireTestFile.Error())
	})

	t.Run("All valid", func(t *testing.T) {
		distributedPods := int32(2)
		spec.DistributedPods = &distributedPods
		spec.TestFile = "my-test"
		err := jmeter.TransformLoadTestSpec(spec)
		assert.NoError(t, err)
		assert.Equal(t, spec.MasterConfig.Image, "master-image")
		assert.Equal(t, spec.MasterConfig.Tag, "master-tag")
		assert.Equal(t, spec.WorkerConfig.Image, "worker-image")
		assert.Equal(t, spec.WorkerConfig.Tag, "worker-tag")
	})
}
