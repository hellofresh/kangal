package k6

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

	scriptTestFileName = "test.js"
	testdataFileName   = "testdata"
)

var (
	loadTestLabelKey         = "app"
	loadTestWorkerLabelValue = "loadtest-worker-pod"

	defaultArgs = []string{"run", "/data/test.js"}
)

// NewJob creates a new job that runs k6
func (b *Backend) NewJob(
	loadTest loadTestV1.LoadTest,
	volumes []coreV1.Volume,
	mounts []coreV1.VolumeMount,
	envvarSecret *coreV1.Secret,
	reportURL string,
	index int32,
) *batchV1.Job {
	logger := b.logger.With(
		zap.String("loadtest", loadTest.GetName()),
		zap.String("namespace", loadTest.Status.Namespace),
	)

	ownerRef := metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest"))

	imageRef := loadTest.Spec.MasterConfig
	if imageRef == ":" {
		imageRef = b.image
		logger.Warn("Loadtest.Spec.MasterConfig is empty; using default image", zap.String("imageRef", string(imageRef)))
	}

	var envVars []coreV1.EnvVar
	if reportURL != "" {
		envVars = append(envVars, coreV1.EnvVar{
			Name:  "REPORT_PRESIGNED_URL",
			Value: reportURL,
		})
	}

	var envFrom []coreV1.EnvFromSource
	if envvarSecret != nil {
		envFrom = append(envFrom, coreV1.EnvFromSource{
			SecretRef: &coreV1.SecretEnvSource{
				LocalObjectReference: coreV1.LocalObjectReference{
					Name: newSecretName(loadTest),
				},
			},
		})
	}

	args := make([]string, len(defaultArgs))
	copy(args, defaultArgs)

	if loadTest.Spec.Duration != 0 {
		args = append(args, "--duration", loadTest.Spec.Duration.String())
	}
	if *loadTest.Spec.DistributedPods > 1 {
		args = append(args, segmentArgs(index, *loadTest.Spec.DistributedPods)...)
		args = append(args, sequenceArgs(*loadTest.Spec.DistributedPods)...)
	}

	backoffLimit := int32(0)
	distributedPod := int32(1)
	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jobName(index),
			Namespace: loadTest.Status.Namespace,
			Labels: map[string]string{
				"name":           loadTestJobName,
				loadTestLabelKey: loadTestWorkerLabelValue,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		Spec: batchV1.JobSpec{
			Parallelism:  &distributedPod,
			Completions:  &distributedPod,
			BackoffLimit: &backoffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"name":           loadTestJobName,
						loadTestLabelKey: loadTestWorkerLabelValue,
					},
					Annotations: b.podAnnotations,
				},
				Spec: coreV1.PodSpec{
					NodeSelector: b.nodeSelector,
					Tolerations:  b.podTolerations,
					Affinity: &coreV1.Affinity{
						PodAntiAffinity: &coreV1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []coreV1.WeightedPodAffinityTerm{
								{
									Weight: 1,
									PodAffinityTerm: coreV1.PodAffinityTerm{
										LabelSelector: &metaV1.LabelSelector{
											MatchLabels: map[string]string{
												loadTestLabelKey: loadTestWorkerLabelValue,
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					RestartPolicy: "Never",
					Volumes:       volumes,
					Containers: []coreV1.Container{
						{
							Name:         "k6",
							Image:        string(imageRef),
							Env:          envVars,
							Resources:    backends.BuildResourceRequirements(b.resources),
							Args:         args,
							VolumeMounts: mounts,
							EnvFrom:      envFrom,
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

func segmentArgs(index, total int32) []string {
	args := make([]string, 0)
	args = append(args, "--execution-segment")
	var segment strings.Builder
	switch {
	case index == 0:
		segment.WriteString("0")
	default:
		segment.WriteString(fmt.Sprintf("%d/%d", index, total))
	}
	segment.WriteString(":")
	index++
	switch {
	case index == total:
		segment.WriteString("1")
	default:
		segment.WriteString(fmt.Sprintf("%d/%d", index, total))
	}
	return append(args, segment.String())
}

func sequenceArgs(total int32) []string {
	args := make([]string, 0)
	args = append(args, "--execution-segment-sequence")

	seq := []string{"0"}

	for i := int32(1); i < total; i++ {
		seq = append(seq, fmt.Sprintf("%d/%d", i, total))
	}

	seq = append(seq, "1")

	return append(args, strings.Join(seq[:], ","))
}

// determineLoadTestPhaseFromJobs reads existing job statuses and determines what the loadtest status should be
func determineLoadTestPhaseFromJobs(jobs []batchV1.Job) loadTestV1.LoadTestPhase {
	for _, job := range jobs {
		if job.Status.Failed > int32(0) {
			return loadTestV1.LoadTestErrored
		}
		if job.Status.Active > int32(0) {
			return loadTestV1.LoadTestRunning
		}
		if job.Status.Succeeded == 0 && job.Status.Failed == 0 {
			return loadTestV1.LoadTestStarting
		}
	}
	return loadTestV1.LoadTestFinished
}

func jobName(index int32) string {
	return fmt.Sprintf("%s-%d", loadTestJobName, index)
}

func determineLoadTestStatusFromJobs(jobs []batchV1.Job) batchV1.JobStatus {
	for _, job := range jobs {
		if job.Status.Failed > int32(0) {
			return job.Status
		}
	}
	for _, job := range jobs {
		if job.Status.Active > int32(0) {
			return job.Status
		}
	}

	return jobs[0].Status
}

func newSecretName(loadTest loadTestV1.LoadTest) string {
	return fmt.Sprintf("%s-envvar", loadTest.ObjectMeta.Name)
}

func newSecret(loadTest loadTestV1.LoadTest, envs map[string]string) *coreV1.Secret {
	name := newSecretName(loadTest)

	ownerRef := metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest"))

	return &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				loadTestLabelKey: name,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		StringData: envs,
	}
}
