package locust

import (
	"fmt"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/hellofresh/kangal/pkg/backends"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	loadTestLabelKey         = "app"
	loadTestMasterLabelValue = "loadtest-master"
	loadTestWorkerLabelValue = "loadtest-worker-pod"
)

func newConfigMapName(loadTest loadTestV1.LoadTest) string {
	return fmt.Sprintf("%s-testfile", loadTest.ObjectMeta.Name)
}

func newConfigMap(loadTest loadTestV1.LoadTest) *coreV1.ConfigMap {
	name := newConfigMapName(loadTest)

	ownerRef := metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest"))

	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       loadTest.Status.Namespace,
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		Data: map[string]string{
			"locustfile.py": loadTest.Spec.TestFile,
		},
	}
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

func newMasterJobName(loadTest loadTestV1.LoadTest) string {
	return fmt.Sprintf("%s-master", loadTest.ObjectMeta.Name)
}

func newMasterJob(
	loadTest loadTestV1.LoadTest,
	testfileConfigMap *coreV1.ConfigMap,
	envvarSecret *coreV1.Secret,
	reportURL string,
	masterResources backends.Resources,
	podAnnotations map[string]string,
	nodeSelector map[string]string,
	podTolerations []coreV1.Toleration,
	image loadTestV1.ImageDetails,
	logger *zap.Logger,
) *batchV1.Job {
	name := newMasterJobName(loadTest)

	ownerRef := metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest"))

	imageRef := fmt.Sprintf("%s:%s", image.Image, image.Tag)
	if imageRef == ":" {
		imageRef = fmt.Sprintf("%s:%s", loadTest.Spec.MasterConfig.Image, loadTest.Spec.MasterConfig.Tag)
		logger.Warn("Loadtest.Spec.MasterConfig is empty; using default image", zap.String("imageRef", imageRef))
	}

	envVars := []coreV1.EnvVar{
		{Name: "LOCUST_HEADLESS", Value: "true"},
		{Name: "LOCUST_MODE_MASTER", Value: "true"},
		{Name: "LOCUST_EXPECT_WORKERS", Value: fmt.Sprintf("%d", *loadTest.Spec.DistributedPods)},
		{Name: "LOCUST_LOCUSTFILE", Value: "/data/locustfile.py"},
		{Name: "LOCUST_CSV", Value: "/tmp/report"},
		{Name: "LOCUST_HOST", Value: loadTest.Spec.TargetURL},
		{Name: "LOCUST_RUN_TIME", Value: loadTest.Spec.Duration.String()},
	}

	if "" != reportURL {
		envVars = append(envVars, coreV1.EnvVar{
			Name:  "REPORT_PRESIGNED_URL",
			Value: reportURL,
		})
	}

	envFrom := make([]coreV1.EnvFromSource, 0)
	if envvarSecret != nil {
		envFrom = append(envFrom, coreV1.EnvFromSource{
			SecretRef: &coreV1.SecretEnvSource{
				LocalObjectReference: coreV1.LocalObjectReference{
					Name: newSecretName(loadTest),
				},
			},
		})
	}

	// Locust does not support recovering after a failure
	backoffLimit := int32(0)

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: loadTest.Status.Namespace,
			Labels: map[string]string{
				"name":           name,
				loadTestLabelKey: loadTestMasterLabelValue,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"name":           name,
						loadTestLabelKey: loadTestMasterLabelValue,
					},
					Annotations: podAnnotations,
				},
				Spec: coreV1.PodSpec{
					NodeSelector:  nodeSelector,
					Tolerations:   podTolerations,
					RestartPolicy: "Never",
					Containers: []coreV1.Container{
						{
							Name:            "locust",
							Image:           imageRef,
							ImagePullPolicy: "Always",
							Env:             envVars,
							VolumeMounts: []coreV1.VolumeMount{
								{
									Name:      "testfile",
									MountPath: "/data/locustfile.py",
									SubPath:   "locustfile.py",
								},
							},
							Resources: backends.BuildResourceRequirements(masterResources),
							EnvFrom:   envFrom,
						},
					},
					Volumes: []coreV1.Volume{
						{
							Name: "testfile",
							VolumeSource: coreV1.VolumeSource{
								ConfigMap: &coreV1.ConfigMapVolumeSource{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: testfileConfigMap.GetName(),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newMasterService(loadTest loadTestV1.LoadTest, masterJob *batchV1.Job) *coreV1.Service {
	name := fmt.Sprintf("%s-master", loadTest.ObjectMeta.Name)

	ownerRef := metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest"))

	return &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: loadTest.Status.Namespace,
			Labels: map[string]string{
				loadTestLabelKey: name,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		Spec: coreV1.ServiceSpec{
			Selector:  masterJob.Spec.Template.Labels,
			ClusterIP: "None",
			Ports: []coreV1.ServicePort{
				{
					Name: "server",
					Port: 5557,
					TargetPort: intstr.IntOrString{
						IntVal: 5557,
					},
				},
			},
		},
	}
}

func newWorkerJobName(loadTest loadTestV1.LoadTest) string {
	return fmt.Sprintf("%s-worker", loadTest.ObjectMeta.Name)
}

func newWorkerJob(
	loadTest loadTestV1.LoadTest,
	testfileConfigMap *coreV1.ConfigMap,
	envvarSecret *coreV1.Secret,
	masterService *coreV1.Service,
	workerResources backends.Resources,
	podAnnotations map[string]string,
	nodeSelector map[string]string,
	podTolerations []coreV1.Toleration,
	image loadTestV1.ImageDetails,
	logger *zap.Logger,
) *batchV1.Job {
	name := newWorkerJobName(loadTest)

	ownerRef := metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest"))

	imageRef := fmt.Sprintf("%s:%s", image.Image, image.Tag)
	if imageRef == ":" {
		imageRef = fmt.Sprintf("%s:%s", loadTest.Spec.MasterConfig.Image, loadTest.Spec.MasterConfig.Tag)
		logger.Warn("Loadtest.Spec.MasterConfig is empty; using default image", zap.String("imageRef", imageRef))
	}

	envVars := []coreV1.EnvVar{
		{Name: "LOCUST_MODE_WORKER", Value: "true"},
		{Name: "LOCUST_LOCUSTFILE", Value: "/data/locustfile.py"},
		{Name: "LOCUST_MASTER_NODE_HOST", Value: masterService.GetName()},
		{Name: "LOCUST_MASTER_NODE_PORT", Value: "5557"},
	}

	envFrom := make([]coreV1.EnvFromSource, 0)
	if envvarSecret != nil {
		envFrom = append(envFrom, coreV1.EnvFromSource{
			SecretRef: &coreV1.SecretEnvSource{
				LocalObjectReference: coreV1.LocalObjectReference{
					Name: newSecretName(loadTest),
				},
			},
		})
	}

	// Locust does not support recovering after a failure
	backoffLimit := int32(0)

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: loadTest.Status.Namespace,
			Labels: map[string]string{
				loadTestLabelKey: name,
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
						loadTestLabelKey: loadTestWorkerLabelValue,
					},
					Annotations: podAnnotations,
				},
				Spec: coreV1.PodSpec{
					NodeSelector:  nodeSelector,
					Tolerations:   podTolerations,
					RestartPolicy: "Never",
					Containers: []coreV1.Container{
						{
							Name:            "locust",
							Image:           imageRef,
							ImagePullPolicy: "Always",
							Env:             envVars,
							VolumeMounts: []coreV1.VolumeMount{
								{
									Name:      "testfile",
									MountPath: "/data/locustfile.py",
									SubPath:   "locustfile.py",
								},
							},
							Resources: backends.BuildResourceRequirements(workerResources),
							EnvFrom:   envFrom,
						},
					},
					Volumes: []coreV1.Volume{
						{
							Name: "testfile",
							VolumeSource: coreV1.VolumeSource{
								ConfigMap: &coreV1.ConfigMapVolumeSource{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: testfileConfigMap.GetName(),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// determineLoadTestStatusFromJobs reads existing job statuses and determines what the loadtest status should be
func determineLoadTestStatusFromJobs(masterJob *batchV1.Job, workerJob *batchV1.Job) loadTestV1.LoadTestPhase {
	if workerJob.Status.Failed > int32(0) || masterJob.Status.Failed > int32(0) {
		return loadTestV1.LoadTestErrored
	}

	if workerJob.Status.Active > int32(0) || masterJob.Status.Active > int32(0) {
		return loadTestV1.LoadTestRunning
	}

	if workerJob.Status.Succeeded == 0 && workerJob.Status.Failed == 0 {
		return loadTestV1.LoadTestStarting
	}
	if masterJob.Status.Succeeded == 0 && masterJob.Status.Failed == 0 {
		return loadTestV1.LoadTestStarting
	}

	return loadTestV1.LoadTestFinished
}
