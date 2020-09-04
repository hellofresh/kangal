package cmd

import (
	"reflect"
	"testing"

	"github.com/hellofresh/kangal/pkg/controller"
)

func TestControllerPopulateCfgFromOpts(t *testing.T) {
	type fields struct {
		kubeConfig           string
		masterURL            string
		namespaceAnnotations []string
		podAnnotations       []string
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
			}
			if got, _ := populateCfgFromOpts(controller.Config{}, opts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("populateCfgFromOpts() = %v, want %v", got, tt.want)
			}
		})
	}
}
