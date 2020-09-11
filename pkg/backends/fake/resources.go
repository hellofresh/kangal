package fake

import (
	"fmt"

	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

// NewMasterJob creates a new job which runs the Fake master pod
func (c *Fake) NewMasterJob() *batchV1.Job {
	// For fake provider we don't really create load test and just use alpine image with some sleep
	// to simulate load test job. Please don't use Fake provider in production.
	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-master",
			Labels: map[string]string{
				"app": "loadtest-master",
			},
			OwnerReferences: []metaV1.OwnerReference{
				*metaV1.NewControllerRef(c.loadTest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest")),
			},
		},
		Spec: batchV1.JobSpec{
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"app": "loadtest-master",
					},
				},
				Spec: coreV1.PodSpec{
					RestartPolicy: "Never",
					Containers: []coreV1.Container{
						{
							Name:            "loadtest-master",
							Image:           fmt.Sprintf("%s:%s", c.loadTest.Spec.MasterConfig.Image, c.loadTest.Spec.MasterConfig.Tag),
							ImagePullPolicy: "Always",
							Command:         []string{"/bin/sh", "-c", "--"},
							Args:            []string{"sleep 10"},
						},
					},
				},
			},
		},
	}
}
