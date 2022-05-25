package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseToleration(t *testing.T) {
	type expected struct {
		toleration Toleration
		err        string
	}
	tests := []struct {
		name       string
		toleration string
		expect     expected
	}{
		{
			name:       "valid toleration",
			toleration: "key:value:Equal:NoSchedule",
			expect: expected{
				toleration: Toleration{
					Key:      "key",
					Value:    "value",
					Operator: "Equal",
					Effect:   "NoSchedule",
				},
			},
		},
		{
			name:       "empty toleration",
			toleration: "",
			expect: expected{
				err: `Failed to parse toleration, expected pattern "key:value:Operation:Effect"`,
			},
		},
		{
			name:       "incomplete toleration",
			toleration: "key:value:Equal",
			expect: expected{
				err: `Failed to parse toleration, expected pattern "key:value:Operation:Effect"`,
			},
		},
		{
			name:       "invalid operator",
			toleration: "key:value:Invalid:NoSchedule",
			expect: expected{
				err: `Invalid operator type "Invalid"`,
			},
		},
		{
			name:       "invalid effect",
			toleration: "key:value:Equal:Invalid",
			expect: expected{
				err: `Invalid effect type "Invalid"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseToleration(tt.toleration)
			if tt.expect.err != "" {
				assert.EqualError(t, err, tt.expect.err)
			}
			assert.Equal(t, tt.expect.toleration, actual)
		})
	}
}
