package v1

//NewSpec initialize spec for LoadTest custom resource
func NewSpec(loadTestType LoadTestType, overwrite bool, distributedPods int32, testFileStr, testDataStr, envVarsStr string, masterConfig, workerConfig ImageDetails) LoadTestSpec {
	lt := LoadTestSpec{
		Type:            loadTestType,
		Overwrite:       overwrite,
		MasterConfig:    masterConfig,
		WorkerConfig:    workerConfig,
		DistributedPods: &distributedPods,
		TestFile:        testFileStr,
		TestData:        testDataStr,
		EnvVars:         envVarsStr,
	}
	return lt
}
