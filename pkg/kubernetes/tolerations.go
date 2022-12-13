package kubernetes

import (
	"errors"
	"fmt"
	"strings"

	kubeCoreV1 "k8s.io/api/core/v1"
)

// Toleration is a representation of the Kubernetes toleration
type Toleration struct {
	Key      string
	Value    string
	Operator string
	Effect   string
}

// Tolerations is an alias to Toleration slice
type Tolerations []Toleration

// ParseToleration parses a pattern of key:value:Operation:Effect to Toleration
func ParseToleration(toleration string) (Toleration, error) {
	tolerationParts := strings.Split(toleration, ":")
	if len(tolerationParts) != 4 {
		return Toleration{}, errors.New(`failed to parse toleration, expected pattern "key:value:Operation:Effect"`)
	}

	t := Toleration{
		Key:      tolerationParts[0],
		Value:    tolerationParts[1],
		Operator: tolerationParts[2],
		Effect:   tolerationParts[3],
	}

	if err := t.Validate(); err != nil {
		return Toleration{}, err
	}

	return t, nil
}

// ParseTolerations parses a csv pattern of key:value:Operation:Effect to Tolerations
func ParseTolerations(tolerations []string) (Tolerations, error) {
	var err error
	parsedTolerations := make(Tolerations, len(tolerations))
	for i, toleration := range tolerations {
		parsedTolerations[i], err = ParseToleration(toleration)
		if err != nil {
			return nil, err
		}
	}
	return parsedTolerations, nil
}

// Validate validates the Toleration properties to be compatible with Kubernetes Tolerations
func (t Toleration) Validate() error {
	op := kubeCoreV1.TolerationOperator(t.Operator)
	effect := kubeCoreV1.TaintEffect(t.Effect)

	if op != kubeCoreV1.TolerationOpEqual && op != kubeCoreV1.TolerationOpExists {
		return fmt.Errorf("invalid operator type %q", op)
	}

	if effect != kubeCoreV1.TaintEffectNoExecute &&
		effect != kubeCoreV1.TaintEffectNoSchedule &&
		effect != kubeCoreV1.TaintEffectPreferNoSchedule {
		return fmt.Errorf("invalid effect type %q", effect)
	}

	return nil
}

// KubeToleration maps Tolerations to  Kubenetes Tolerations
func (tolerations Tolerations) KubeToleration() []kubeCoreV1.Toleration {
	kubeTolerations := make([]kubeCoreV1.Toleration, len(tolerations))
	for i, toleration := range tolerations {
		kubeTolerations[i] = kubeCoreV1.Toleration{
			Key:      toleration.Key,
			Value:    toleration.Value,
			Operator: kubeCoreV1.TolerationOperator(toleration.Operator),
			Effect:   kubeCoreV1.TaintEffect(toleration.Effect),
		}
	}
	return kubeTolerations
}
