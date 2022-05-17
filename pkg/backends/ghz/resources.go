package ghz

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hellofresh/kangal/pkg/backends"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	loadTestJobName           = "loadtest-job"
	loadTestFileConfigMapName = "loadtest-testfile"
	loadTestDataConfigMapName = "loadtest-testdata"
	loadTestFileVolumeName    = "loadtest-testfile-volume"
	loadTestDataVolumeName    = "loadtest-testdata-volume"

	configFileName   = "config"
	testdataFileName = "testdata.protoset"
)

var defaultArgs = []string{
	"--config=/data/config",
	"--output=/results",
	"--format=html",
}

// NewJob creates a new job that runs ghz
func (b *Backend) NewJob(
	loadTest loadTestV1.LoadTest,
	volumes []coreV1.Volume,
	mounts []coreV1.VolumeMount,
	reportURL string,
) *batchV1.Job {
	logger := b.logger.With(
		zap.String("loadtest", loadTest.GetName()),
		zap.String("namespace", loadTest.Status.Namespace),
	)

	ownerRef := metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest"))

	imageRef := fmt.Sprintf("%s:%s", loadTest.Spec.MasterConfig.Image, loadTest.Spec.MasterConfig.Tag)
	if imageRef == ":" {
		imageRef = fmt.Sprintf("%s:%s", b.image.Image, b.image.Tag)
		logger.Warn("Loadtest.Spec.MasterConfig is empty; using default image", zap.String("imageRef", imageRef))
	}

	envVars := []coreV1.EnvVar{}
	if "" != reportURL {
		envVars = append(envVars, coreV1.EnvVar{
			Name:  "REPORT_PRESIGNED_URL",
			Value: reportURL,
		})
	}

	backoffLimit := int32(0)

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      loadTestJobName,
			Namespace: loadTest.Status.Namespace,
			Labels: map[string]string{
				"name": loadTestJobName,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		Spec: batchV1.JobSpec{
			Parallelism:  loadTest.Spec.DistributedPods,
			Completions:  loadTest.Spec.DistributedPods,
			BackoffLimit: &backoffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"name": loadTestJobName,
					},
					Annotations: b.podAnnotations,
				},
				Spec: coreV1.PodSpec{
					NodeSelector:  b.nodeSelector,
					RestartPolicy: "Never",
					Volumes:       volumes,
					Containers: []coreV1.Container{
						{
							Name:         "ghz",
							Image:        imageRef,
							Env:          envVars,
							Resources:    backends.BuildResourceRequirements(b.resources),
							Args:         defaultArgs,
							VolumeMounts: mounts,
						},
					},
				},
			},
		},
	}
}

// NewFileVolumeAndMount creates a new volume and volume mount for a configmap file
func NewFileVolumeAndMount(name, cfg, filename string) (coreV1.Volume, coreV1.VolumeMount) {
	v := coreV1.Volume{
		Name: name,
		VolumeSource: coreV1.VolumeSource{
			ConfigMap: &coreV1.ConfigMapVolumeSource{
				LocalObjectReference: coreV1.LocalObjectReference{
					Name: cfg,
				},
			},
		},
	}

	m := coreV1.VolumeMount{
		Name:      name,
		MountPath: fmt.Sprintf("/data/%s", filename),
		SubPath:   filename,
	}

	return v, m
}

// NewFileConfigMap creates a configmap for the provided file information
func NewFileConfigMap(cfgName, filename, content string) (*coreV1.ConfigMap, error) {
	if strings.TrimSpace(cfgName) == "" {
		return nil, errors.New("empty config name")
	}

	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("invalid name for configmap %s", cfgName)
	}

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("invalid file %s for configmap %s, empty content", filename, cfgName)
	}

	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name: cfgName,
		},
		Data: map[string]string{
			filename: content,
		},
	}, nil
}

// determineLoadTestStatusFromJobs reads existing job statuses and determines what the loadtest status should be
func determineLoadTestStatusFromJobs(job *batchV1.Job) loadTestV1.LoadTestPhase {
	if job.Status.Failed > int32(0) {
		return loadTestV1.LoadTestErrored
	}

	if job.Status.Active > int32(0) {
		return loadTestV1.LoadTestRunning
	}

	if job.Status.Succeeded == 0 && job.Status.Failed == 0 {
		return loadTestV1.LoadTestStarting
	}

	return loadTestV1.LoadTestFinished
}
