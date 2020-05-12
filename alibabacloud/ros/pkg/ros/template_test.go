package ros

import (
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/appconf"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestNewTemplate(t *testing.T) {
	type args struct {
		appContext    *appconf.Context
		appConf       *appconf.AppConf
		compSchematic *v1alpha1.ComponentSchematic
	}
	tests := []struct {
		name string
		args args
		want *Template
	}{
		{
			name: "TestNormal",
			args: args{
				appContext: &appconf.Context{DryRun: true},
				appConf: &appconf.AppConf{
					TypeMeta: v1.TypeMeta{},
					ObjectMeta: v1.ObjectMeta{
						Namespace: "MyNamespace",
					},
					Spec: v1alpha1.ApplicationConfigurationSpec{
						Components: []v1alpha1.ComponentConfiguration{
							{
								InstanceName:  "Vpc",
								ComponentName: "VpcComp",
							},
						},
					},
					Status: v1alpha1.ApplicationConfigurationStatus{},
				},
				compSchematic: &v1alpha1.ComponentSchematic{
					Spec: v1alpha1.ComponentSpec{
						WorkloadType: "ros.aliyun.com/v1alpha1.ECS_VPC",
						WorkloadSettings: runtime.RawExtension{
							Raw: []byte(`{"VpcName": "MyVpc", "CidrBlock": "192.168.0.0/16", "Description": "My VPC"}`),
						}},
				},
			},
			want: &Template{
				ROSTemplateFormatVersion: "2015-09-01",
				Parameters:               map[string]Parameter{},
				Resources: map[string]Resource{
					"Vpc": {
						Type: "ALIYUN::ECS::VPC",
						Properties: map[string]interface{}{
							"VpcName":     "MyVpc",
							"CidrBlock":   "192.168.0.0/16",
							"Description": "My VPC",
						},
						DependsOn:      []string{},
						DeletionPolicy: "Retain",
					}},
				Outputs: map[string]Output{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, _ := NewTemplate(
				tt.args.appContext,
				tt.args.appConf,
				WithCompSchematicGetter(func(namespace, name string) (*v1alpha1.ComponentSchematic, error) {
					return tt.args.compSchematic, nil
				}))
			assert.Equal(t, tt.want, template)
		})
	}
}

func TestTemplate_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		template *Template
		want     string
	}{
		{
			name: "TestNormal",
			template: &Template{
				ROSTemplateFormatVersion: "2015-09-01",
				Description:              "Create a VPC",
				Parameters: map[string]Parameter{
					"VpcName": {
						Type:        "String",
						Default:     "MyVpc",
						Description: "My VPC",
					},
				},
				Resources: map[string]Resource{
					"Vpc": {
						Type: "ALIYUN::ECS::VPC",
						Properties: map[string]interface{}{
							"VpcName": map[string]string{
								"Ref": "VpcName",
							},
							"CidrBlock": "192.168.0.0/16",
						},
						DependsOn:      nil,
						DeletionPolicy: "",
					},
				},
				Outputs: map[string]Output{
					"VpcId": {
						Description: "vpc id",
						Value: map[string]string{
							"Ref": "Vpc",
						},
					},
				},
			},
			want: `{"ROSTemplateFormatVersion":"2015-09-01","Description":"Create a VPC","Parameters":{"VpcName":{"Type":"String","Default":"MyVpc","Description":"My VPC"}},"Resources":{"Vpc":{"Type":"ALIYUN::ECS::VPC","Properties":{"CidrBlock":"192.168.0.0/16","VpcName":{"Ref":"VpcName"}}}},"Outputs":{"VpcId":{"Description":"vpc id","Value":{"Ref":"Vpc"}}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, _ := tt.template.Marshal()

			assert.Equal(t, tt.want, string(content))
		})
	}
}

func TestTemplate_genResource(t *testing.T) {
	type args struct {
		resourceType string
		compConf     v1alpha1.ComponentConfiguration
		compSpec     v1alpha1.ComponentSpec
		compConfs    []v1alpha1.ComponentConfiguration
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]Resource
		wantErr bool
	}{
		{
			name: "TestBasic",
			args: args{
				resourceType: "ALIYUN::ECS::VPC",
				compSpec: v1alpha1.ComponentSpec{
					WorkloadSettings: runtime.RawExtension{
						Raw: []byte(`{"VpcName": "MyVpc", "CidrBlock": "192.168.0.0/16", "Description": "My VPC"}`)}},
				compConf: v1alpha1.ComponentConfiguration{
					InstanceName: "Vpc",
					ParameterValues: []v1alpha1.ParameterValue{
						{
							Name:  "VpcName",
							Value: "newvpc",
						},
					},
				},
				compConfs: []v1alpha1.ComponentConfiguration{},
			},
			want: map[string]Resource{
				"Vpc": {
					Type: "ALIYUN::ECS::VPC",
					Properties: map[string]interface{}{
						"VpcName":     "newvpc",
						"CidrBlock":   "192.168.0.0/16",
						"Description": "My VPC",
					},
					DependsOn:      []string{},
					DeletionPolicy: "Retain",
				},
			},
		},
		{
			name: "TestRefAnotherComp",
			args: args{
				resourceType: "ALIYUN::ECS::VPC",
				compSpec: v1alpha1.ComponentSpec{
					WorkloadSettings: runtime.RawExtension{
						Raw: []byte(`{"VpcName": "MyVpc", "CidrBlock": "192.168.0.0/16", "Description": "My VPC"}`)}},
				compConf: v1alpha1.ComponentConfiguration{
					InstanceName: "Vpc",
					ParameterValues: []v1alpha1.ParameterValue{
						{
							Name: "ResourceGroupId",
							From: &v1alpha1.ParameterFrom{
								Component: "Res",
								FieldPath: ".status.ResourceGroupId",
							},
						},
					},
				},
				compConfs: []v1alpha1.ComponentConfiguration{
					{
						InstanceName: "Vpc",
					},
					{
						InstanceName: "Res",
					},
				},
			},
			want: map[string]Resource{
				"Vpc": {
					Type: "ALIYUN::ECS::VPC",
					Properties: map[string]interface{}{
						"VpcName":     "MyVpc",
						"CidrBlock":   "192.168.0.0/16",
						"Description": "My VPC",
						"ResourceGroupId": map[string][]string{
							"Fn::GetAtt": {"Res", "ResourceGroupId"}},
					},
					DependsOn:      []string{"Res"},
					DeletionPolicy: "Retain",
				},
			},
		},
		{
			name: "TestDeletePolicy",
			args: args{
				resourceType: "ALIYUN::ECS::VPC",
				compSpec: v1alpha1.ComponentSpec{
					WorkloadSettings: runtime.RawExtension{
						Raw: []byte(`{"VpcName": "MyVpc", "CidrBlock": "192.168.0.0/16", "Description": "My VPC"}`)}},
				compConf: v1alpha1.ComponentConfiguration{
					InstanceName: "Vpc",
					Traits: []v1alpha1.TraitBinding{
						{
							Name: "DeletionPolicy",
							Properties: runtime.RawExtension{
								Raw: []byte(`{"policy": "Delete"}`),
							},
						},
					},
				},
				compConfs: []v1alpha1.ComponentConfiguration{},
			},
			want: map[string]Resource{
				"Vpc": {
					Type: "ALIYUN::ECS::VPC",
					Properties: map[string]interface{}{
						"VpcName":     "MyVpc",
						"CidrBlock":   "192.168.0.0/16",
						"Description": "My VPC",
					},
					DependsOn:      []string{},
					DeletionPolicy: "Delete",
				},
			},
		},
		{
			name: "TestRetainPolicy",
			args: args{
				resourceType: "ALIYUN::ECS::VPC",
				compSpec: v1alpha1.ComponentSpec{
					WorkloadSettings: runtime.RawExtension{
						Raw: []byte(`{"VpcName": "MyVpc", "CidrBlock": "192.168.0.0/16", "Description": "My VPC"}`)}},
				compConf: v1alpha1.ComponentConfiguration{
					InstanceName: "Vpc",
					Traits: []v1alpha1.TraitBinding{
						{
							Name: "DeletionPolicy",
							Properties: runtime.RawExtension{
								Raw: []byte(`{"policy": "NotDeleteWillBeRetain"}`),
							},
						},
					},
				},
				compConfs: []v1alpha1.ComponentConfiguration{},
			},
			want: map[string]Resource{
				"Vpc": {
					Type: "ALIYUN::ECS::VPC",
					Properties: map[string]interface{}{
						"VpcName":     "MyVpc",
						"CidrBlock":   "192.168.0.0/16",
						"Description": "My VPC",
					},
					DependsOn:      []string{},
					DeletionPolicy: "Retain",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, _ := NewTemplate(&appconf.Context{DryRun: true}, &appconf.AppConf{})
			err := template.genResource(tt.args.resourceType, tt.args.compConf, tt.args.compSpec, tt.args.compConfs)
			assert.Nil(t, err)
			assert.Equal(t, tt.want, template.Resources)
		})
	}
}

func TestTemplate_genOutputs(t *testing.T) {
	type args struct {
		instanceName       string
		resourceAttributes map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]Output
	}{
		{
			name: "TestNormal",
			args: args{
				instanceName: "Vpc",
				resourceAttributes: map[string]interface{}{
					"VpcId": map[string]interface{}{},
					"VRouterId": map[string]interface{}{
						"Description": "Router id of created VPC.",
					},
				},
			},
			want: map[string]Output{
				"Vpc.VpcId": {
					Value:       map[string][2]string{"Fn::GetAtt": {"Vpc", "VpcId"}},
					Description: "",
				},
				"Vpc.VRouterId": {
					Value:       map[string][2]string{"Fn::GetAtt": {"Vpc", "VRouterId"}},
					Description: "Router id of created VPC.",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, _ := NewTemplate(&appconf.Context{DryRun: true}, &appconf.AppConf{})
			template.genOutputs(tt.args.instanceName, tt.args.resourceAttributes)
			assert.Equal(t, tt.want, template.Outputs)
		})
	}
}

func Test_getResourceType(t *testing.T) {
	type args struct {
		workloadType string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "TestNormal",
			args:    args{workloadType: "ros.aliyun.com/v1alpha1.ECS_VPC"},
			want:    "ALIYUN::ECS::VPC",
			wantErr: false,
		},
		{
			name:    "TestFormatError",
			args:    args{workloadType: "invalidformat"},
			want:    "workloadType must be format of {group}/{version}.{type}",
			wantErr: true,
		},
		{
			name:    "TestFormatError1",
			args:    args{workloadType: "invalid"},
			want:    "workloadType must be format of {group}/{version}.{type}",
			wantErr: true,
		},
		{
			name:    "TestVersionTypeError2",
			args:    args{workloadType: "ros.aliyun.com/invalid"},
			want:    "workloadType must be format of {group}/{version}.{type}",
			wantErr: true,
		},
		{
			name:    "TestVersionTypeError2",
			args:    args{workloadType: "ros.aliyun.com/a.b.c"},
			want:    "workloadType must be format of {group}/{version}.{type}",
			wantErr: true,
		},
		{
			name:    "TestGroupError",
			args:    args{workloadType: "invalid/ECS.VPC"},
			want:    "Group 'invalid' in workloadType is not supported. Support group: ros.aliyun.com",
			wantErr: true,
		},
		{
			name:    "TestVersionError",
			args:    args{workloadType: "ros.aliyun.com/invalid.ECS_VPC"},
			want:    "Version 'invalid' in workloadType is not supported. Support version: v1alpha1",
			wantErr: true,
		},
		{
			name:    "TestTypeError",
			args:    args{workloadType: "ros.aliyun.com/v1alpha1.VPC"},
			want:    "Type 'VPC' in workloadType must be format of {product}_{restype}",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getResourceType(tt.args.workloadType)
			if tt.wantErr {
				assert.NotNil(t, err)
				assert.Equal(t, tt.want, err.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
