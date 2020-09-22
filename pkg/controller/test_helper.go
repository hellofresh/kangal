package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typeV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	watchtools "k8s.io/client-go/tools/watch"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

// CreateLoadtest creates a load test CR
func CreateLoadtest(clientSet clientSetV.Clientset, pods int32, name, testFile, testData, envVars string, loadTestType apisLoadTestV1.LoadTestType) error {
	var ev, td = "", ""
	tf, err := readFile(testFile)
	if err != nil {
		return err
	}
	if testData != "" {
		td, err = readFile(testData)
		if err != nil {
			return err
		}
	}

	if envVars != "" {
		ev, err = readFile(envVars)
		if err != nil {
			return err
		}
	}

	ltObj := &apisLoadTestV1.LoadTest{}
	ltObj.Name = name
	ltObj.Spec.DistributedPods = &pods
	ltObj.Spec.Type = loadTestType
	ltObj.Spec.TestFile = tf
	ltObj.Spec.EnvVars = ev
	ltObj.Spec.TestData = td

	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

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

func waitLoadtestFunc(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Added:
		return true, nil
	case watch.Modified:
		return true, nil
	default:
		return false, nil
	}
}

// WaitLoadtest waits until Loadtest resources exists
func WaitLoadtest(clientSet clientSetV.Clientset, loadtestName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	watchObj, err := clientSet.KangalV1().LoadTests().Watch(ctx, metaV1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", loadtestName),
	})
	if err != nil {
		return err
	}

	_, err = watchtools.UntilWithoutRetry(ctx, watchObj, waitLoadtestFunc)

	return err
}

// DeleteLoadTest deletes a load test CR
func DeleteLoadTest(clientSet clientSetV.Clientset, loadtestName string, testname string) error {
	fmt.Printf("Deleting object %v for the test %v \n", loadtestName, testname)
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	if err := clientSet.KangalV1().LoadTests().Delete(ctx, loadtestName, metaV1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

// GetLoadtest returns a load test name
func GetLoadtest(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return result.Name, nil
}

// GetLoadtestTestdata returns a load test name
func GetLoadtestTestdata(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return result.Spec.TestData, nil
}

// GetLoadtests returns a list of load tests
func GetLoadtests(clientSet clientSetV.Clientset) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	result, err := clientSet.KangalV1().LoadTests().List(ctx, metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	lts := make([]string, len(result.Items))
	for i := range result.Items {
		lts = append(lts, result.Items[i].Name)
	}
	return lts, nil
}

// GetLoadtestEnvVars returns a load test name
func GetLoadtestEnvVars(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return result.Spec.EnvVars, nil
}

// GetLoadtestNamespace returns a load test namespace
func GetLoadtestNamespace(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return result.Status.Namespace, nil
}

// GetLoadtestPhase returns the current phase of given loadtest
func GetLoadtestPhase(clientSet clientSetV.Clientset, loadtestName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	result, err := clientSet.KangalV1().LoadTests().Get(ctx, loadtestName, metaV1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(result.Status.Phase), nil
}

// GetDistributedPods returns a number of distributed pods in load test namespace
func GetDistributedPods(clientSet typeV1.CoreV1Interface, namespace string) (coreV1.PodList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	opts := metaV1.ListOptions{
		LabelSelector: "app=loadtest-worker-pod",
	}
	pods, err := clientSet.Pods(namespace).List(ctx, opts)
	if err != nil {
		return coreV1.PodList{}, err
	}
	return *pods, nil
}

// GetMasterPod returns a master pod in load test namespace
func GetMasterPod(clientSet typeV1.CoreV1Interface, namespace string) (coreV1.PodList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

	opts := metaV1.ListOptions{
		LabelSelector: "app=loadtest-master",
	}
	master, err := clientSet.Pods(namespace).List(ctx, opts)
	if err != nil {
		return coreV1.PodList{}, err
	}
	return *master, nil
}

// GetSecret returns a list of created secrets according to the given label
func GetSecret(clientSet typeV1.CoreV1Interface, namespace string) (coreV1.SecretList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), KubeTimeout)
	defer cancel()

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
	return config, nil
}

// WaitForResource sleeps to wait kubernetes resources to be created
func WaitForResource(d time.Duration) {
	time.Sleep(d * time.Second)
}
