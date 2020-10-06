package locust

import (
	"fmt"
	"net/url"

	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	defaultBackoffLimit  int32 = 1
	defaultExpectWorkers int32 = 1
)

func newConfigMap(loadTest *loadtestV1.LoadTest) *coreV1.ConfigMap {
	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name: fmt.Sprintf("%s-testfile", loadTest.ObjectMeta.Name),
		},
		Data: map[string]string{
			"locustfile.py": loadTest.Spec.TestFile,
		},
	}
}

func newMasterJob(loadTest *loadtestV1.LoadTest, preSignedURL *url.URL, podAnnotations map[string]string) *batchV1.Job {
	expectWorkers := defaultExpectWorkers
	if nil != loadTest.Spec.DistributedPods {
		expectWorkers = *loadTest.Spec.DistributedPods
	}

	envVars := []coreV1.EnvVar{
		{
			Name:  "LOCUST_HEADLESS",
			Value: "true",
		},
		{
			Name:  "LOCUST_MODE_MASTER",
			Value: "true",
		},
		{
			Name:  "LOCUST_EXPECT_WORKERS",
			Value: fmt.Sprintf("%d", expectWorkers),
		},
		{
			Name:  "LOCUST_LOCUSTFILE",
			Value: "/data/locustfile.py",
		},
		{
			Name:  "LOCUST_CSV",
			Value: "/tmp/",
		},
		{
			Name:  "LOCUST_HOST",
			Value: "https://httpdump.io/ezigh",
		},
	}

	if nil != preSignedURL {
		envVars = append(envVars, coreV1.EnvVar{
			Name:  "REPORT_PRESIGNED_URL",
			Value: preSignedURL.String(),
		})
	}

	ownerRef := metaV1.NewControllerRef(loadTest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest"))
	job := fmt.Sprintf("%s-master", loadTest.ObjectMeta.Name)
	testfileConfigMap := fmt.Sprintf("%s-testfile", loadTest.ObjectMeta.Name)
	image := fmt.Sprintf("%s:%s", loadTest.Spec.MasterConfig.Image, loadTest.Spec.MasterConfig.Tag)
	// image = "ubuntu:latest"

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name: job,
			Labels: map[string]string{
				"app": job,
			},
			OwnerReferences: []metaV1.OwnerReference{*ownerRef},
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &defaultBackoffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"app": job,
					},
					Annotations: podAnnotations,
				},
				Spec: coreV1.PodSpec{
					RestartPolicy: "Never",
					Containers: []coreV1.Container{
						{
							Name:            "locust",
							Image:           image,
							ImagePullPolicy: "Always",
							Env:             envVars,
							// Command:         []string{"tail", "-f", "/dev/null"},
							VolumeMounts: []coreV1.VolumeMount{
								{
									Name:      "testfile",
									MountPath: "/data/locustfile.py",
									SubPath:   "locustfile.py",
								},
							},
							Resources: coreV1.ResourceRequirements{
								Requests: map[coreV1.ResourceName]resource.Quantity{
									coreV1.ResourceMemory: resource.MustParse("1Gi"),
									coreV1.ResourceCPU:    resource.MustParse("500m"),
								},
								Limits: map[coreV1.ResourceName]resource.Quantity{
									coreV1.ResourceMemory: resource.MustParse("4Gi"),
									coreV1.ResourceCPU:    resource.MustParse("2000m"),
								},
							},
						},
					},
					Volumes: []coreV1.Volume{
						{
							Name: "testfile",
							VolumeSource: coreV1.VolumeSource{
								ConfigMap: &coreV1.ConfigMapVolumeSource{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: testfileConfigMap,
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
