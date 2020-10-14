package jmeter

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/hellofresh/kangal/pkg/core/helper"
	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	// LoadTestLabel label used for JMeter resources
	LoadTestLabel = "loadtest"
	// TestFileHash is a load test label name for keeping loadtest file hash
	TestFileHash = "test-file-hash"
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
	//loadTestSecretLabels is a labeles set for created secrets
	loadTestSecretLabels = map[string]string{
		loadTestSecretLabelKey: loadTestSecretLabel,
	}
)

// NewJMeterSettingsConfigMap creates a new configmap which holds jmeter config
func (c *JMeter) NewJMeterSettingsConfigMap() *coreV1.ConfigMap {
	data := map[string]string{
		"jmeter.properties": `num_sample_threshold=5
time_threshold=1000
#---------------------------------------------------------------------------
# Results file configuration
#---------------------------------------------------------------------------
jmeter.save.saveservice.output_format=csv
jmeter.save.saveservice.response_data=true
jmeter.save.saveservice.response_data.on_error=true
jmeter.save.saveservice.response_message=true
#---------------------------------------------------------------------------
# Additional property files to load
#---------------------------------------------------------------------------
user.properties=user.properties
system.properties=system.properties
#---------------------------------------------------------------------------
# Reporting configuration
#---------------------------------------------------------------------------
jmeter.reportgenerator.apdex_satisfied_threshold=200
jmeter.reportgenerator.apdex_tolerated_threshold=500
jmeter.reportgenerator.report_title=Kangal JMeter Dashboard
jmeter.reportgenerator.overall_granularity=10000
jmeter.save.saveservice.timestamp_format = yyyy/MM/dd HH:mm:ss zzz`,
	}

	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name: LoadTestLabel,
			Labels: map[string]string{
				"app": "hf-jmeter",
			},
		},
		Data: data,
	}
}

// NewConfigMap creates a new configMap containing loadtest script
func (c *JMeter) NewConfigMap() *coreV1.ConfigMap {
	loadtest := c.loadTest
	testfile := loadtest.Spec.TestFile

	data := map[string]string{
		"testfile.jmx": testfile,
	}

	return &coreV1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name: loadTestFile,
		},
		Data: data,
	}
}

// NewSecret creates a secret from file envVars
func (c *JMeter) NewSecret() (*coreV1.Secret, error) {
	loadtest := c.loadTest
	envVars := loadtest.Spec.EnvVars

	secretMap, err := helper.ReadEnvs(envVars)
	if err != nil {
		c.logger.Error("Error on creating secrets from envVars file", zap.Error(err))
		return nil, err
	}

	return &coreV1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   loadTestEnvVars,
			Labels: loadTestSecretLabels,
		},
		StringData: secretMap,
	}, nil
}

// NewTestdataConfigMap creates a new configMap containing testdata
func (c *JMeter) NewTestdataConfigMap() ([]*coreV1.ConfigMap, error) {
	loadtest := c.loadTest
	testdata := loadtest.Spec.TestData
	n := int(*loadtest.Spec.DistributedPods)

	cMaps := make([]*coreV1.ConfigMap, n)

	chunks, err := splitTestData(testdata, n, c.logger)
	if err != nil {
		c.logger.Error("Error on splitting csv test data", zap.Error(err))
		return nil, err
	}

	stringWriter := new(strings.Builder)

	for i := 0; i < n; i++ {
		stringWriter.Reset()
		csvWriter := csv.NewWriter(stringWriter)
		if err := csvWriter.WriteAll(chunks[i]); err != nil {
			c.logger.Error("Error on writing csv test data to chunks", zap.Error(err))
			return nil, err
		}

		data := map[string]string{
			"testdata.csv": stringWriter.String(),
		}

		cmName := fmt.Sprintf("%s-%03d", loadTestFile, i)

		cMaps[i] = &coreV1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name: cmName,
			},
			Data: data,
		}
	}

	return cMaps, nil
}

// NewPod creates a new pod which mounts a configmap that contains jmeter testdata
func (c *JMeter) NewPod(i int, configMap *coreV1.ConfigMap, podAnnotations map[string]string) *coreV1.Pod {
	loadtest := c.loadTest
	optionalVolume := true
	WorkerConfig := loadtest.Spec.WorkerConfig

	return &coreV1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%03d", loadTestWorkerName, i),
			Labels:      loadTestWorkerPodLabels,
			Annotations: podAnnotations,
			OwnerReferences: []metaV1.OwnerReference{
				*metaV1.NewControllerRef(loadtest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest")),
			},
		},
		Spec: coreV1.PodSpec{
			Containers: []coreV1.Container{
				{
					Name:            loadTestWorkerName,
					Image:           fmt.Sprintf("%s:%s", WorkerConfig.Image, WorkerConfig.Tag),
					ImagePullPolicy: "Always",
					Ports: []coreV1.ContainerPort{
						{ContainerPort: 1099},
						{ContainerPort: 50000},
					},
					VolumeMounts: []coreV1.VolumeMount{
						{
							Name:      "testdata",
							MountPath: "/testdata",
						},
					},
					Resources: helper.BuildResourceRequirements(c.workerResources),
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
			Volumes: []coreV1.Volume{
				{
					Name: "testdata",
					VolumeSource: coreV1.VolumeSource{
						ConfigMap: &coreV1.ConfigMapVolumeSource{
							LocalObjectReference: coreV1.LocalObjectReference{
								Name: configMap.Name,
							},
							Optional: &optionalVolume,
						},
					},
				},
			},
		},
	}
}

// NewJMeterMasterJob creates a new job which runs the jmeter master pod
func (c *JMeter) NewJMeterMasterJob(preSignedURL *url.URL, podAnnotations map[string]string) *batchV1.Job {
	loadtest := c.loadTest
	var one int32 = 1
	MasterConfig := loadtest.Spec.MasterConfig

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

	if nil != preSignedURL {
		jMeterEnvVars = append(jMeterEnvVars, coreV1.EnvVar{
			Name:  "REPORT_PRESIGNED_URL",
			Value: preSignedURL.String(),
		})
	}

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name: loadTestJobName,
			Labels: map[string]string{
				loadTestMasterJobLabelKey: loadTestJobName,
			},
			OwnerReferences: []metaV1.OwnerReference{
				*metaV1.NewControllerRef(loadtest, loadtestV1.SchemeGroupVersion.WithKind("LoadTest")),
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
					RestartPolicy: "Never",
					Containers: []coreV1.Container{
						{
							Name:            loadTestJobName,
							Image:           fmt.Sprintf("%s:%s", MasterConfig.Image, MasterConfig.Tag),
							ImagePullPolicy: "Always",
							Env:             jMeterEnvVars,
							VolumeMounts: []coreV1.VolumeMount{
								{
									Name:      "tests",
									MountPath: "/tests",
								},
								{
									Name:      "config",
									MountPath: "/opt/apache-jmeter-5.0/bin/jmeter.properties",
									SubPath:   "jmeter.properties",
								},
							},
							Resources: helper.BuildResourceRequirements(c.masterResources),
						},
					},
					Volumes: []coreV1.Volume{
						{
							Name: "tests",
							VolumeSource: coreV1.VolumeSource{
								ConfigMap: &coreV1.ConfigMapVolumeSource{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: loadTestFile,
									},
								},
							},
						},
						{
							Name: "config",
							VolumeSource: coreV1.VolumeSource{
								ConfigMap: &coreV1.ConfigMapVolumeSource{
									LocalObjectReference: coreV1.LocalObjectReference{
										Name: LoadTestLabel,
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
func (c *JMeter) NewJMeterService() *coreV1.Service {
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

// SplitTestData splits provided csv test data and returns the array of file chunks
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
