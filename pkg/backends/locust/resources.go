package locust

import (
	"fmt"

	"go.uber.org/zap"

	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/hellofresh/kangal/pkg/core/helper"
	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	loadTestMasterLabelKey   = "app"
	loadTestMasterLabelValue = "loadtest-master"
)

func newConfigMapName(loadTest *loadtestV1.LoadTest) string {
	return fmt.Sprintf("%s-testfile", loadTest.ObjectMeta.Name)
}

func newConfigMap(loadTest *loadtestV1.LoadTest) *coreV1.ConfigMap {
	name := newConfigMapName(loadTest)

	ownerRef := metaV1.NewControllerRef(loadTest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest"))

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

func newSecretName(loadTest *loadtestV1.LoadTest) string {
	return fmt.Sprintf("%s-envvar", loadTest.ObjectMeta.Name)
}

func newSecret(loadTest *loadtestV1.LoadTest, envs map[string]string) *coreV1.Secret {
	name := newSecretName(loadTest)

	ownerRef := metaV1.NewControllerRef(loadTest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest"))

	return &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": name,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		StringData: envs,
	}
}

func newMasterJobName(loadTest *loadtestV1.LoadTest) string {
	return fmt.Sprintf("%s-master", loadTest.ObjectMeta.Name)
}

func newMasterJob(loadTest *loadtestV1.LoadTest, testfileConfigMap *coreV1.ConfigMap, envvarSecret *coreV1.Secret, reportURL string, masterResources helper.Resources, podAnnotations map[string]string, imageName, imageTag string, logger *zap.Logger) *batchV1.Job {
	name := newMasterJobName(loadTest)

	ownerRef := metaV1.NewControllerRef(loadTest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest"))

	imageRef := fmt.Sprintf("%s:%s", loadTest.Spec.MasterConfig.Image, loadTest.Spec.MasterConfig.Tag)
	if imageRef == "" {
		imageRef = fmt.Sprintf("%s:%s", imageName, imageTag)
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

	envFrom := []coreV1.EnvFromSource{}
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
				"name":                 name,
				loadTestMasterLabelKey: loadTestMasterLabelValue,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"name":                 name,
						loadTestMasterLabelKey: loadTestMasterLabelValue,
					},
					Annotations: podAnnotations,
				},
				Spec: coreV1.PodSpec{
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
							Resources: helper.BuildResourceRequirements(masterResources),
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

func newMasterService(loadTest *loadtestV1.LoadTest, masterJob *batchV1.Job) *coreV1.Service {
	name := fmt.Sprintf("%s-master", loadTest.ObjectMeta.Name)

	ownerRef := metaV1.NewControllerRef(loadTest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest"))

	return &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: loadTest.Status.Namespace,
			Labels: map[string]string{
				"app": name,
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

func newWorkerJobName(loadTest *loadtestV1.LoadTest) string {
	return fmt.Sprintf("%s-worker", loadTest.ObjectMeta.Name)
}

func newWorkerJob(loadTest *loadtestV1.LoadTest, testfileConfigMap *coreV1.ConfigMap, envvarSecret *coreV1.Secret, masterService *coreV1.Service, workerResources helper.Resources, podAnnotations map[string]string, imageName, imageTag string, logger *zap.Logger) *batchV1.Job {
	name := newWorkerJobName(loadTest)

	ownerRef := metaV1.NewControllerRef(loadTest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest"))

	imageRef := fmt.Sprintf("%s:%s", loadTest.Spec.MasterConfig.Image, loadTest.Spec.MasterConfig.Tag)
	if imageRef == "" {
		imageRef = fmt.Sprintf("%s:%s", imageName, imageTag)
		logger.Warn("Loadtest.Spec.MasterConfig is empty; using default image", zap.String("imageRef", imageRef))
	}

	envVars := []coreV1.EnvVar{
		{Name: "LOCUST_MODE_WORKER", Value: "true"},
		{Name: "LOCUST_LOCUSTFILE", Value: "/data/locustfile.py"},
		{Name: "LOCUST_MASTER_NODE_HOST", Value: masterService.GetName()},
		{Name: "LOCUST_MASTER_NODE_PORT", Value: "5557"},
	}

	envFrom := []coreV1.EnvFromSource{}
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
				"app": name,
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
						"app": name,
					},
					Annotations: podAnnotations,
				},
				Spec: coreV1.PodSpec{
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
							Resources: helper.BuildResourceRequirements(workerResources),
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

func getLoadTestStatusFromJobs(masterJob *batchV1.Job, workerJob *batchV1.Job) loadtestV1.LoadTestPhase {
	if workerJob.Status.Failed > int32(0) || masterJob.Status.Failed > int32(0) {
		return loadtestV1.LoadTestErrored
	}

	if workerJob.Status.Active > int32(0) || masterJob.Status.Active > int32(0) {
		return loadtestV1.LoadTestRunning
	}

	if workerJob.Status.Succeeded == 0 && workerJob.Status.Failed == 0 {
		return loadtestV1.LoadTestStarting
	}
	if masterJob.Status.Succeeded == 0 && masterJob.Status.Failed == 0 {
		return loadtestV1.LoadTestStarting
	}

	return loadtestV1.LoadTestFinished
}
