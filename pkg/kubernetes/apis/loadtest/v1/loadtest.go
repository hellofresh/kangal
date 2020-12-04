package v1

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/technosophos/moniker"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	maxTagLength = 63 // K8s limit.
)

// Possible load test errors
var (
	ErrUnknownLoadTestPhase = errors.New("unknown Load Test phase")
)

//BuildLoadTestObject initialize new LoadTest custom resource
func BuildLoadTestObject(spec LoadTestSpec) (*LoadTest, error) {
	generatedName := moniker.New().NameSep("-")

	name := "loadtest-" + generatedName

	labels := map[string]string{
		"test-file-hash": getHashFromString(spec.TestFile),
	}

	for tagName, tagValue := range spec.Tags {
		tagName = fmt.Sprintf("test-tag-%s", tagName)
		labels[tagName] = tagValue
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

// LoadTestTagsFromString builds tags from string.
func LoadTestTagsFromString(tagsStr string) (LoadTestTags, error) {
	if tagsStr == "" {
		return LoadTestTags{}, nil
	}

	pairs := strings.Split(strings.TrimSpace(tagsStr), ",")
	tags := make(LoadTestTags, len(pairs))

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if len(pair) < 1 {
			continue
		}

		parts := strings.SplitN(pair, ":", 2)
		label := strings.TrimSpace(parts[0])

		if len(label) == 0 {
			return nil, ErrTagMissingLabel
		}

		if len(parts) != 2 {
			return nil, ErrTagMissingValue
		}

		value := strings.TrimSpace(parts[1])

		if len(value) == 0 {
			return nil, ErrTagMissingValue
		}

		if len(value) > maxTagLength {
			return nil, ErrTagValueMaxLengthExceeded
		}

		tags[label] = value
	}

	return tags, nil
}

// LoadTestPhaseFromString tries to get LoadTestPhase from string value.
// Empty phase is a valid value and does not cause error, so caller should take care of checking if the phase is set
// to one of the pre-defined values or empty.
func LoadTestPhaseFromString(phase string) (LoadTestPhase, error) {
	switch LoadTestPhase(strings.ToLower(phase)) {
	case "":
		return "", nil
	case LoadTestCreating:
		return LoadTestCreating, nil
	case LoadTestStarting:
		return LoadTestStarting, nil
	case LoadTestRunning:
		return LoadTestRunning, nil
	case LoadTestFinished:
		return LoadTestFinished, nil
	case LoadTestErrored:
		return LoadTestErrored, nil
	}

	return "", ErrUnknownLoadTestPhase
}

func getHashFromString(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}
