package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hellofresh/kangal/pkg/controller"
)

func TestControllerPopulateCfgFromOpts(t *testing.T) {
	type fields struct {
		kubeConfig           string
		masterURL            string
		namespaceAnnotations []string
		podAnnotations       []string
		nodeSelectors        []string
	}
	tests := []struct {
		name   string
		fields fields
		want   controller.Config
	}{
		{
			name:   "test with empty annotations",
			fields: fields{},
			want: controller.Config{
				NamespaceAnnotations: map[string]string{},
				PodAnnotations:       map[string]string{},
				NodeSelectors:        map[string]string{},
			},
		},
		{
			name: "test with aws annotations",
			fields: fields{
				namespaceAnnotations: []string{"iam.amazonaws.com/permitted:.*"},
				podAnnotations:       []string{"iam.amazonaws.com/role:arn:aws:iam::someid:role/some-role-name"},
			},
			want: controller.Config{
				NamespaceAnnotations: map[string]string{"iam.amazonaws.com/permitted": ".*"},
				PodAnnotations:       map[string]string{"iam.amazonaws.com/role": "arn:aws:iam::someid:role/some-role-name"},
				NodeSelectors:        map[string]string{},
			},
		},
		{
			name: `test with node selectors`,
			fields: fields{
				nodeSelectors: []string{`nodelabel:"test"`},
			},
			want: controller.Config{
				NamespaceAnnotations: map[string]string{},
				PodAnnotations:       map[string]string{},
				NodeSelectors:        map[string]string{"nodelabel": "test"},
			},
		},
		{
			name: `test with some "`,
			fields: fields{
				namespaceAnnotations: []string{`iam.amazonaws.com/permitted:".*"`},
				podAnnotations:       []string{`iam.amazonaws.com/role:arn:aws:iam::"someid:role/some-role-name"`},
			},
			want: controller.Config{
				NamespaceAnnotations: map[string]string{"iam.amazonaws.com/permitted": ".*"},
				PodAnnotations:       map[string]string{"iam.amazonaws.com/role": "arn:aws:iam::someid:role/some-role-name"},
				NodeSelectors:        map[string]string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &controllerCmdOptions{
				kubeConfig:           tt.fields.kubeConfig,
				masterURL:            tt.fields.masterURL,
				namespaceAnnotations: tt.fields.namespaceAnnotations,
				podAnnotations:       tt.fields.podAnnotations,
				nodeSelectors:        tt.fields.nodeSelectors,
			}
			got, _ := populateCfgFromOpts(controller.Config{}, opts)
			assert.EqualValues(t, tt.want, got)
		})
	}
}
