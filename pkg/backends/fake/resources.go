package fake

import (
	batchV1 "k8s.io/api/batch/v1"
)

// NewMasterJob creates a new job which runs the Fake master pod
func (c *Fake) NewMasterJob() *batchV1.Job {
	return c.backend.newMasterJob(*c.loadTest)
}
