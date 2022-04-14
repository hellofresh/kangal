/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadTest is a specification for a LoadTest resource
type LoadTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadTestSpec   `json:"spec"`
	Status LoadTestStatus `json:"status"`
}

// LoadTestPodsStatus is a specification for a LoadTestStatus resource
type LoadTestPodsStatus struct {
	Current *int32 `json:"current"`
	Desired *int32 `json:"desired"`
}

// LoadTestSpec is the spec for a LoadTest resource
type LoadTestSpec struct {
	Type            LoadTestType      `json:"type"`
	Overwrite       bool              `json:"overwrite"`
	MasterConfig    ImageDetails      `json:"masterConfig"`
	WorkerConfig    ImageDetails      `json:"workerConfig"`
	DistributedPods *int32            `json:"distributedPods"`
	Tags            LoadTestTags      `json:"tags"`
	TestFile        []byte            `json:"testFile"`
	TestData        []byte            `json:"testData,omitempty"`
	EnvVars         map[string]string `json:"envVars,omitempty"`
	TargetURL       string            `json:"targetURL,omitempty"`
	Duration        time.Duration     `json:"duration,omitempty"`
}

// LoadTestTags is a list of tags of a LoadTest resource.
type LoadTestTags map[string]string

// MasterConfig is the configuration information for each resource type
type MasterConfig struct {
	Master *ImageDetails
}

// WorkerConfig is the configuration information for each resource type
type WorkerConfig struct {
	Worker *ImageDetails
}

// ImageDetails is the image information for a resource
type ImageDetails struct {
	Image string `json:"image"`
	Tag   string `json:"tag"`
}

// LoadTestStatus is the status for a LoadTest resource
type LoadTestStatus struct {
	Phase     LoadTestPhase      `json:"phase"`
	Namespace string             `json:"namespace"`
	JobStatus batchv1.JobStatus  `json:"jobStatus"`
	Pods      LoadTestPodsStatus `json:"pods"`
}

// LoadTestPhase defines the phases that a loadtest can be in
type LoadTestPhase string

// String returns string representation of LoadTestPhase
func (p LoadTestPhase) String() string {
	return string(p)
}

const (
	// LoadTestCreating is after a namespaces has been created for a LoadTest
	// but before any process have been created
	LoadTestCreating LoadTestPhase = "creating"
	// LoadTestStarting is when we have created the jmeter job but there are
	// no active pods
	LoadTestStarting LoadTestPhase = "starting"
	// LoadTestRunning is the status that a loadtest has when the jmeter master job
	// has at least 1 active pod
	LoadTestRunning LoadTestPhase = "running"
	// LoadTestFinished is used when the jmeter master job has run and is finished
	// running. This does not tell us of the status of the job, to know if the job
	// was successful we need to look at the jobStatus
	LoadTestFinished LoadTestPhase = "finished"
	// LoadTestErrored is set in case of resource creating failed because of
	// incorrect data provided by user
	LoadTestErrored LoadTestPhase = "errored"
)

// LoadTestType needs to be specified to know what tool to use when running a loadtest
type LoadTestType string

// String returns string representation of LoadTestType
func (t LoadTestType) String() string {
	return string(t)
}

const (
	// LoadTestTypeJMeter tells the controller to run the loadtest using JMeter
	LoadTestTypeJMeter LoadTestType = "JMeter"
	// LoadTestTypeFake tells controller to use fake provider
	LoadTestTypeFake LoadTestType = "Fake"
	// LoadTestTypeLocust tells controller to use Locust provider
	LoadTestTypeLocust LoadTestType = "Locust"
	// LoadTestTypeGhz tells controller to use ghz provider
	LoadTestTypeGhz LoadTestType = "Ghz"
	// LoadTestTypeK6 tells controller to use k6 provider
	LoadTestTypeK6 LoadTestType = "K6"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadTestList is a list of LoadTest resources
type LoadTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []LoadTest `json:"items"`
}
