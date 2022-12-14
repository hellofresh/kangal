package jmeter

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/hellofresh/kangal/pkg/backends"
	"github.com/hellofresh/kangal/pkg/core/waitfor"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	// LoadTestLabel label used for JMeter resources
	LoadTestLabel = "loadtest"
	// loadTestJobName is the name of the job that runs loadtests
	loadTestJobName = LoadTestLabel + "-master"
	// loadTestWorkerPodLabelKey key we are using for the worker pod label
	loadTestWorkerPodLabelKey = "app"
	// loadTestWorkerPodLabelValue value we are using for the worker pod label
	loadTestWorkerPodLabelValue = LoadTestLabel + "-worker-pod"
	// loadTestWorkerServiceName is the name of the service for talking to worker pods
	loadTestWorkerServiceName = LoadTestLabel + "-workers"
	// loadTestWorkerName is the base name of the worker pods
	loadTestWorkerName = LoadTestLabel + "-worker"
	// loadTestWorkerRemoteCustomDataVolumeSize is the default size of custom data volume
	loadTestWorkerRemoteCustomDataVolumeSize = "1Gi"
	// loadTestFile is the name of the config map that is used to hold testfile data
	loadTestFile = LoadTestLabel + "-testfile"
	// loadTestMasterJobLabelKey key we are using for the master job label
	loadTestMasterJobLabelKey = "app"
	// loadTestEnvVars is a name of a config map containing environment variables
	loadTestEnvVars = LoadTestLabel + "-env-vars"
	// loadTestSecretLabel is a label of a secret containing environment variables
	loadTestSecretLabel = "env-vars-from-file"
	// loadTestSecretLabelKey is a label key of a secret containing environment variables
	loadTestSecretLabelKey = "secret-source"
)

var (
	// loadTestWorkerPodLabels the labels set on all JMeter worker pods
	loadTestWorkerPodLabels = map[string]string{
		loadTestWorkerPodLabelKey: loadTestWorkerPodLabelValue,
	}
	//loadTestSecretLabels is a labels set for created secrets
	loadTestSecretLabels = map[string]string{
		loadTestSecretLabelKey: loadTestSecretLabel,
	}
)

// NewSecret creates a secret from file envVars
func (b *Backend) NewSecret(loadTest loadTestV1.LoadTest) (*coreV1.Secret, error) {
	secretMap := loadTest.Spec.EnvVars

	return &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   loadTestEnvVars,
			Labels: loadTestSecretLabels,
		},
		StringData: secretMap,
	}, nil
}

// NewPVC creates a new pvc for customdata
func (b *Backend) NewPVC(loadTest loadTestV1.LoadTest, i int) *coreV1.PersistentVolumeClaim {
	volumeSize := loadTestWorkerRemoteCustomDataVolumeSize
	if val, ok := loadTest.Spec.EnvVars["JMETER_WORKER_REMOTE_CUSTOM_DATA_VOLUME_SIZE"]; ok {
		volumeSize = val
	}
	return &coreV1.PersistentVolumeClaim{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   fmt.Sprintf("pvc-%s", loadTestWorkerName),
			Labels: loadTestWorkerPodLabels,
			OwnerReferences: []metaV1.OwnerReference{
				*metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest")),
			},
		},
		Spec: coreV1.PersistentVolumeClaimSpec{
			AccessModes: []coreV1.PersistentVolumeAccessMode{coreV1.ReadWriteMany},
			Resources: coreV1.ResourceRequirements{
				Requests: coreV1.ResourceList{
					coreV1.ResourceName(coreV1.ResourceStorage): resource.MustParse(volumeSize),
				},
			},
		},
	}
}

// NewPod creates a new pod which mounts a configmap that contains jmeter testdata
func (b *Backend) NewPod(loadTest loadTestV1.LoadTest, i int, configMapName string, podAnnotations map[string]string) *coreV1.Pod {
	logger := b.logger.With(
		zap.String("loadtest", loadTest.GetName()),
		zap.String("namespace", loadTest.Status.Namespace),
	)

	optionalVolume := true

	imageRef := fmt.Sprintf("%s:%s", loadTest.Spec.WorkerConfig.Image, loadTest.Spec.WorkerConfig.Tag)
	if loadTest.Spec.WorkerConfig.Image == "" || loadTest.Spec.WorkerConfig.Tag == "" {
		imageRef = fmt.Sprintf("%s:%s", b.workerConfig.Image, b.workerConfig.Tag)
		logger.Debug("Loadtest.Spec.WorkerConfig is empty; using worker image from config", zap.String("imageRef", imageRef))
	}

	volumeMounts := []coreV1.VolumeMount{}
	volumes := []coreV1.Volume{}
	if configMapName != "" {
		volumeMounts = []coreV1.VolumeMount{
			{
				Name:      "testdata",
				MountPath: "/testdata/testdata.csv",
				SubPath:   backends.LoadTestData,
			},
		}

		volumes = []coreV1.Volume{
			{
				Name: "testdata",
				VolumeSource: coreV1.VolumeSource{
					ConfigMap: &coreV1.ConfigMapVolumeSource{
						LocalObjectReference: coreV1.LocalObjectReference{
							Name: configMapName,
						},
						Optional: &optionalVolume,
					},
				},
			},
		}

	}

	pod := &coreV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%03d", loadTestWorkerName, i),
			Labels:      loadTestWorkerPodLabels,
			Annotations: podAnnotations,
			OwnerReferences: []metaV1.OwnerReference{
				*metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest")),
			},
		},
		Spec: coreV1.PodSpec{
			NodeSelector: b.nodeSelector,
			Tolerations:  b.tolerations,
			Affinity: &coreV1.Affinity{
				PodAntiAffinity: &coreV1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []coreV1.WeightedPodAffinityTerm{
						{
							Weight: 1,
							PodAffinityTerm: coreV1.PodAffinityTerm{
								LabelSelector: &metaV1.LabelSelector{
									MatchLabels: loadTestWorkerPodLabels,
								},
								TopologyKey: "kubernetes.io/hostname",
							},
						},
					},
				},
			},
			Containers: []coreV1.Container{
				{
					Name:            loadTestWorkerName,
					Image:           imageRef,
					ImagePullPolicy: "Always",
					Ports: []coreV1.ContainerPort{
						{ContainerPort: 1099},
						{ContainerPort: 50000},
					},
					VolumeMounts: volumeMounts,
					Resources:    backends.BuildResourceRequirements(b.workerResources),
					EnvFrom: []coreV1.EnvFromSource{
						{
							SecretRef: &coreV1.SecretEnvSource{
								LocalObjectReference: coreV1.LocalObjectReference{
									Name: loadTestEnvVars,
								},
							},
						},
					},
				},
			},
			Volumes: volumes,
		},
	}

	if _, ok := loadTest.Spec.EnvVars["JMETER_WORKER_REMOTE_CUSTOM_DATA_ENABLED"]; ok {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, coreV1.VolumeMount{
			Name:      "customdata",
			MountPath: "/customdata",
		})
		pod.Spec.Volumes = append(pod.Spec.Volumes, []coreV1.Volume{
			{
				Name: "customdata",
				VolumeSource: coreV1.VolumeSource{
					PersistentVolumeClaim: &coreV1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("pvc-%s", loadTestWorkerName),
					},
				},
			},
			{
				Name: "rclone-data",
				VolumeSource: coreV1.VolumeSource{
					EmptyDir: &coreV1.EmptyDirVolumeSource{},
				},
			}}...)
		pod.Spec.InitContainers = []coreV1.Container{
			{
				Name:    "get-data",
				Image:   "rclone/rclone:latest",
				Command: []string{"/bin/sh"},
				Args:    []string{"-c", "/usr/local/bin/rclone sync remotecustomdata:$(JMETER_WORKER_REMOTE_CUSTOM_DATA_BUCKET) /customdata || echo \"rsync failed\""},
				VolumeMounts: []coreV1.VolumeMount{
					{
						Name:      "rclone-data",
						MountPath: "/data",
					},
					{
						Name:      "customdata",
						MountPath: "/customdata",
					},
				},
				EnvFrom: []coreV1.EnvFromSource{
					{
						SecretRef: &coreV1.SecretEnvSource{
							LocalObjectReference: coreV1.LocalObjectReference{
								Name: loadTestEnvVars,
							},
						},
					},
				},
			},
		}
	}

	return pod
}

// NewJMeterMasterJob creates a new job which runs the jmeter master pod
func (b *Backend) NewJMeterMasterJob(loadTest loadTestV1.LoadTest, testfileConfigMapName string, reportURL string, podAnnotations map[string]string) *batchV1.Job {
	logger := b.logger.With(
		zap.String("loadtest", loadTest.GetName()),
		zap.String("namespace", loadTest.Status.Namespace),
	)

	var one int32 = 1

	imageRef := fmt.Sprintf("%s:%s", loadTest.Spec.MasterConfig.Image, loadTest.Spec.MasterConfig.Tag)
	if loadTest.Spec.MasterConfig.Image == "" || loadTest.Spec.MasterConfig.Tag == "" {
		imageRef = fmt.Sprintf("%s:%s", b.masterConfig.Image, b.masterConfig.Tag)
		logger.Debug("Loadtest.Spec.MasterConfig is empty; using master image from config", zap.String("imageRef", imageRef))
	}

	jMeterEnvVars := []coreV1.EnvVar{
		{
			Name:  "WORKER_SVC_NAME",
			Value: loadTestWorkerServiceName,
		},
		{
			Name:  "USE_WORKERS",
			Value: "true",
		},
	}

	if reportURL != "" {
		jMeterEnvVars = append(jMeterEnvVars, coreV1.EnvVar{
			Name:  "REPORT_PRESIGNED_URL",
			Value: reportURL,
		})
	}

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name: loadTestJobName,
			Labels: map[string]string{
				loadTestMasterJobLabelKey: loadTestJobName,
			},
			OwnerReferences: []metaV1.OwnerReference{
				*metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest")),
			},
			Annotations: podAnnotations,
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &one,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						loadTestMasterJobLabelKey: loadTestJobName,
					},
					Annotations: podAnnotations,
				},

				Spec: coreV1.PodSpec{
					NodeSelector:  b.nodeSelector,
					Tolerations:   b.tolerations,
					RestartPolicy: "Never",
					Containers: []coreV1.Container{
						{
							Name:            loadTestJobName,
							Image:           imageRef,
							ImagePullPolicy: "Always",
							Env:             jMeterEnvVars,
							VolumeMounts: []coreV1.VolumeMount{
								{
									Name:      "tests",
									MountPath: "/tests/testfile.jmx",
									SubPath:   backends.LoadTestScript,
								},
							},
							Resources: backends.BuildResourceRequirements(b.masterResources),
						},
					},
					Volumes: []coreV1.Volume{
						{
							Name: "tests",
							VolumeSource: coreV1.VolumeSource{
								ConfigMap: &coreV1.ConfigMapVolumeSource{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: testfileConfigMapName,
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

// NewJMeterService creates a new services to talk to jmeter worker pods
func (b *Backend) NewJMeterService() *coreV1.Service {
	return &coreV1.Service{
		ObjectMeta: metaV1.ObjectMeta{
			Name: loadTestWorkerServiceName,
			Labels: map[string]string{
				"app": LoadTestLabel,
			},
		},
		Spec: coreV1.ServiceSpec{
			Selector:  loadTestWorkerPodLabels,
			ClusterIP: "None",
			Ports: []coreV1.ServicePort{
				{
					Name: "server",
					Port: 1099,
					TargetPort: intstr.IntOrString{
						IntVal: 1099,
					},
				},
				{
					Name: "rmi",
					Port: 50000,
					TargetPort: intstr.IntOrString{
						IntVal: 50000,
					},
				},
			},
		},
	}
}

// CreatePodsWithTestdata creates workers Pods
func (b *Backend) CreatePodsWithTestdata(ctx context.Context, configMapNames []string, loadTest *loadTestV1.LoadTest, namespace string) error {
	logger := b.logger.With(
		zap.String("loadtest", loadTest.GetName()),
		zap.String("namespace", loadTest.Status.Namespace),
	)
	for i := 0; i < int(*loadTest.Spec.DistributedPods); i = i + 1 {
		if _, ok := loadTest.Spec.EnvVars["JMETER_WORKER_REMOTE_CUSTOM_DATA_ENABLED"]; ok {
			logger.Info("Remote custom data enabled, creating PVC")

			pvc := b.NewPVC(*loadTest, i)
			_, err := b.kubeClientSet.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metaV1.CreateOptions{})
			if err != nil && !kerrors.IsAlreadyExists(err) {
				logger.Error("Error on creating pvc", zap.Error(err))
				return err
			}

			watchObjPvc, err := b.kubeClientSet.CoreV1().PersistentVolumeClaims(namespace).Watch(ctx, metaV1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", pvc.ObjectMeta.Name),
			})
			if err != nil {
				logger.Warn("unable to watch pvc state", zap.Error(err))
				continue
			}
			waitfor.Resource(watchObjPvc, (waitfor.Condition{}).PvcReady, b.config.WaitForResourceTimeout)
		}

		cmName := ""
		if i < len(configMapNames) {
			cmName = configMapNames[i]
		}
		pod := b.NewPod(*loadTest, i, cmName, b.podAnnotations)
		_, err := b.kubeClientSet.CoreV1().Pods(namespace).Create(ctx, pod, metaV1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			logger.Error("Error on creating distributed pods", zap.Error(err))
			return err
		}

		if kerrors.IsAlreadyExists(err) {
			pod, err = b.kubeClientSet.CoreV1().Pods(namespace).Get(ctx, pod.Name, metaV1.GetOptions{})
			if nil != err {
				logger.Error("unable to reload Pod", zap.Error(err))
				return err
			}
		}

		// JMeter requires all workers to be running before master starts
		// So, wait to pod be running before continue
		watchObj, err := b.kubeClientSet.CoreV1().Pods(namespace).Watch(ctx, metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", pod.ObjectMeta.Name),
		})
		if err != nil {
			logger.Warn("unable to watch pod state", zap.Error(err))
			continue
		}
		waitfor.Resource(watchObj, (waitfor.Condition{}).PodRunning, b.config.WaitForResourceTimeout)
	}
	logger.Info("Created pods with test data")
	return nil
}

// splitTestData splits provided csv test data and returns the array of file chunks
func splitTestData(testdata string, n int, logger *zap.Logger) ([][][]string, error) {
	reader := csv.NewReader(strings.NewReader(testdata))

	count := 0
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("Error on reading testdata csv file", zap.Error(err))
			return nil, err
		}
		count++
	}
	logger.Debug("Testdata file lines count", zap.Int("count", count))

	linesInChunk := count / n
	logger.Debug("Splitting testdata to chunks", zap.Int("linesInChunk", linesInChunk))

	chunk := 0
	chunks := make([][][]string, n)
	reader = csv.NewReader(strings.NewReader(testdata))
	for line := 0; chunk < n; line++ {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if line >= linesInChunk {
			chunk++
			line = 0
		}

		if chunk >= n {
			break
		}

		chunks[chunk] = append(chunks[chunk], rec)
	}
	return chunks, nil
}

func getNamespaceFromLoadTestName(loadTestName string, logger *zap.Logger) (newNamespaceName string, err error) {
	nsName := strings.Split(loadTestName, "-")
	loadTestNameLength := len(nsName)
	if loadTestNameLength < 2 {
		logger.Error("Invalid loadTest name, too short", zap.String("loadTestName", loadTestName), zap.Error(os.ErrInvalid))
		return "", os.ErrInvalid
	}
	newNamespaceName = nsName[loadTestNameLength-2] + "-" + nsName[loadTestNameLength-1]
	return newNamespaceName, nil
}
