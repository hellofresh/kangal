package v1

import (
	"crypto/sha1"
	"encoding/hex"

	"github.com/technosophos/moniker"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//BuildLoadTestObject initialize new LoadTest custom resource
func BuildLoadTestObject(spec LoadTestSpec) (*LoadTest, error) {
	generatedName := moniker.New().NameSep("-")

	name := "loadtest-" + generatedName

	labels := map[string]string{
		"test-file-hash": getHashFromString(spec.TestFile),
	}

	return &LoadTest{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: spec,
		Status: LoadTestStatus{
			Phase: LoadTestCreating,
		},
	}, nil
}

func getHashFromString(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
