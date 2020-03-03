package application

import (
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/ros"
	rosv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAppStackProgressing(t *testing.T) {
	tests := []struct {
		input map[string]string
		exp   bool
	}{
		{
			input: map[string]string{
				AppStackStatus: Progressing,
			},
			exp: true,
		},
		{
			input: map[string]string{
				AppStackStatus: Ready,
			},
			exp: false,
		},
		{
			input: map[string]string{
				AppStackStatus: Failed,
			},
			exp: false,
		},
	}
	for _, ti := range tests {
		got :=IsAppStackProgressing(ti.input)
		assert.Equal(t, ti.exp, got)
	}
}

func TestIsAppStackFailed(t *testing.T) {
	tests := []struct {
		input map[string]string
		exp   bool
	}{
		{
			input: map[string]string{
				AppStackStatus: Progressing,
			},
			exp: false,
		},
		{
			input: map[string]string{
				AppStackStatus: Ready,
			},
			exp: false,
		},
		{
			input: map[string]string{
				AppStackStatus: Failed,
			},
			exp: true,
		},
	}
	for _, ti := range tests {
		got := IsAppStackFailed(ti.input)
		assert.Equal(t, ti.exp, got)
	}
}

func TestGetAppStackSecretName(t *testing.T) {
	tests := []struct {
		input *ros.Context
		exp   string
	}{
		{
			input: &ros.Context{AliUid: "", RegionId: "1342",
				AppConf: &rosv1alpha1.ApplicationConfiguration{ObjectMeta: v1.ObjectMeta{Name: "TestName"}}},
			exp: "testname",
		},
		{
			input: &ros.Context{AliUid: "1545", RegionId: "1342",
				AppConf: &rosv1alpha1.ApplicationConfiguration{ObjectMeta: v1.ObjectMeta{Name: "TestName"}}},
			exp: "1342-1545-testname",
		},
	}
	for _, ti := range tests {
		got := GetAppStackSecretName(ti.input)
		assert.Equal(t, ti.exp, got)
	}
}

func TestGetAppStackOutputSecretName(t *testing.T) {
	tests := []struct {
		input1 *ros.Context
		input2 string
		exp    string
	}{
		{
			input1: &ros.Context{AliUid: "", RegionId: "1342",
				AppConf: &rosv1alpha1.ApplicationConfiguration{ObjectMeta: v1.ObjectMeta{Name: "TestName"}}},
			input2: "TestCompInstanceName",
			exp: "testname-testcompinstancename",
		},
		{
			input1: &ros.Context{AliUid: "1545", RegionId: "1342",
				AppConf: &rosv1alpha1.ApplicationConfiguration{ObjectMeta: v1.ObjectMeta{Name: "TestName"}}},
			input2: "TestCompInstanceName",
			exp: "1342-1545-testname-testcompinstancename",
		},
	}
	for _, ti := range tests {
		got := GetAppStackOutputSecretName(ti.input1, ti.input2)
		assert.Equal(t, ti.exp, got)
	}
}
