package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typeV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/hellofresh/kangal/pkg/backends"
	"github.com/hellofresh/kangal/pkg/core/waitfor"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

const (
	kubeClientDefaultTimeout = 1 * time.Minute
	waitForResourceTimeout   = 30 * time.Second
)

// CreateLoadTest creates a load test CR
func CreateLoadTest(clientSet clientSetV.Clientset, pods int32, name, testFile, testData string, envVars map[string]string, loadTestType apisLoadTestV1.LoadTestType) error {
	var td []byte
	tf, err := os.ReadFile(testFile)
	if err != nil {
		return err
	}
	if testData != "" {
		td, err = os.ReadFile(testData)
		if err != nil {
			return err
		}
	}

	reg := backends.New()

	backend, err := reg.GetBackend(loadTestType)
	if err != nil {
		return err
	}

	loadTestSpec := apisLoadTestV1.LoadTestSpec{
		Type:            loadTestType,
		Overwrite:       false,
		DistributedPods: &pods,
		Tags:            apisLoadTestV1.LoadTestTags{},
		TestFile:        tf,
		TestData:        td,
		EnvVars:         envVars,
		TargetURL:       "",
		Duration:        0,
	}

	err = backend.TransformLoadTestSpec(&loadTestSpec)
	if err != nil {
		return err
	}

	ltObj := &apisLoadTestV1.LoadTest{}
	ltObj.Name = name
	ltObj.Spec = loadTestSpec

	ctx := context.Background()

	_, err = clientSet.KangalV1().LoadTests().Create(ctx, ltObj, metaV1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func readFile(filename string) (string, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	str := string(b)
	return str, nil
}

// WaitLoadTest waits until Loadtest resources exists
func WaitLoadTest(clientSet clientSetV.Clientset, loadtestName string) error {
	watchObj, err := clientSet.KangalV1().LoadTests().Watch(context.Background(), metaV1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", loadtestName),
	})
	if err != nil {
		return err
	}

	_, err = waitfor.Resource(watchObj, (waitfor.Condition{}).LoadTestRunning, waitForResourceTimeout)

	return err
}

// DeleteLoadTest deletes a load test CR
func DeleteLoadTest(clientSet clientSetV.Clientset, loadtestName string, testname string) error {
	fmt.Printf("Deleting object %v for the test %v \n", loadtestName, testname)
	ctx := context.Background()

	if err := clientSet.KangalV1().LoadTests().Delete(ctx, loadtestName, metaV1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// GetLoadTest returns a load test name
func GetLoadTest(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx := context.Background()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return result.Name, nil
}

// GetLoadTestTestdata returns a load test name
func GetLoadTestTestdata(clientSet clientSetV.Clientset, loadtestName string) ([]byte, error) {
	ctx := context.Background()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return result.Spec.TestData, nil
}

// GetLoadTestLabels returns load test labels.
func GetLoadTestLabels(clientSet clientSetV.Clientset, loadtestName string) (map[string]string, error) {
	ctx := context.Background()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return result.Labels, nil
}

// GetLoadTestEnvVars returns a load test name
func GetLoadTestEnvVars(clientSet clientSetV.Clientset, loadtestName string) (map[string]string, error) {
	ctx := context.Background()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return result.Spec.EnvVars, nil
}

// GetLoadTestNamespace returns a load test namespace
func GetLoadTestNamespace(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx := context.Background()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return result.Status.Namespace, nil
}

// GetLoadTestPhase returns the current phase of given loadtest
func GetLoadTestPhase(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx := context.Background()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}

	return result.Status.Phase.String(), nil
}

// GetDistributedPods returns a number of distributed pods in load test namespace
func GetDistributedPods(clientSet typeV1.CoreV1Interface, namespace string) (coreV1.PodList, error) {
	ctx := context.Background()

	opts := metaV1.ListOptions{
		LabelSelector: "app=loadtest-worker-pod",
	}
	pods, err := clientSet.Pods(namespace).List(ctx, opts)
	if err != nil {
		return coreV1.PodList{}, err
	}
	return *pods, nil
}

// GetSecret returns a list of created secrets according to the given label
func GetSecret(clientSet typeV1.CoreV1Interface, namespace string) (coreV1.SecretList, error) {
	ctx := context.Background()

	opts := metaV1.ListOptions{
		LabelSelector: "secret-source=env-vars-from-file",
	}
	secrets, err := clientSet.Secrets(namespace).List(ctx, opts)
	if err != nil {
		return coreV1.SecretList{}, err
	}
	return *secrets, nil
}

// BuildConfig builds a config from the file
func BuildConfig() (*rest.Config, error) {
	homeDir := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", homeDir)
	if err != nil {
		return nil, err
	}
	config.Timeout = kubeClientDefaultTimeout
	return config, nil
}
