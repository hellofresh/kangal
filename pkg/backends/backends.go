package backends

import (
	"github.com/hellofresh/kangal/pkg/backends/fake"
	"github.com/hellofresh/kangal/pkg/backends/jmeter"
	"github.com/hellofresh/kangal/pkg/backends/locust"
)

func init() {
	register(&fake.Fake{})
	register(&jmeter.JMeter{})
	register(&locust.Locust{})
}
