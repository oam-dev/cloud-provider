package k8s

import (
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestNewSecret(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name     string
		args     args
		wantName string
	}{
		{
			name:     "TestUpperName",
			args:     args{name: "MySecret"},
			wantName: "mysecret",
		},
		{
			name:     "TestLowerName",
			args:     args{name: "mysecret"},
			wantName: "mysecret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := NewSecret(tt.args.name)
			assert.Equal(t, tt.wantName, secret.GetName())
		})
	}
}

func TestSecret_GetData(t *testing.T) {
	type args struct {
		name      string
		clientSet *testclient.Clientset
	}
	tests := []struct {
		name     string
		args     args
		wantData map[string]string
		wantErr  error
	}{
		{
			name: "TestGetExist",
			args: args{
				name: "MySecret",
				clientSet: testclient.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: config.RosCtrlConf.Namespace,
							Name:      "mysecret",
						},
						Data: map[string][]byte{"k": []byte("v")},
					},
				),
			},
			wantData: map[string]string{"k": "v"},
			wantErr:  nil,
		},
		{
			name: "TestGetNotExit",
			args: args{
				name:      "NotExist",
				clientSet: testclient.NewSimpleClientset(),
			},
			wantData: map[string]string{},
			wantErr:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSecret(tt.args.name, WithClientSet(tt.args.clientSet))
			data, err := c.GetData()
			assert.Equal(t, tt.wantData, data)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestSecret_UpdateData(t *testing.T) {
	type args struct {
		name      string
		clientSet *testclient.Clientset
		data      map[string]string
	}
	tests := []struct {
		name     string
		args     args
		wantData map[string]string
	}{
		{
			name: "TestUpdateExist",
			args: args{
				name: "MySecret",
				clientSet: testclient.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: config.RosCtrlConf.Namespace,
							Name:      "mysecret",
						},
						Data: map[string][]byte{"k1": []byte("v1")},
					},
				),
				data: map[string]string{"k2": "v2"},
			},
			wantData: map[string]string{"k1": "v1", "k2": "v2"},
		},
		{
			name: "TestUpdateNotExist",
			args: args{
				name:      "MySecret",
				clientSet: testclient.NewSimpleClientset(),
				data:      map[string]string{"k2": "v2"},
			},
			wantData: map[string]string{"k2": "v2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSecret(tt.args.name, WithClientSet(tt.args.clientSet))
			err := c.UpdateData(tt.args.data)
			assert.Nil(t, err)

			data, _ := c.GetData()
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func TestSecret_SetData(t *testing.T) {
	type args struct {
		name      string
		clientSet *testclient.Clientset
		data      map[string]string
	}
	tests := []struct {
		name     string
		args     args
		wantData map[string]string
	}{
		{
			name: "TestSetExist",
			args: args{
				name: "MySecret",
				clientSet: testclient.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: config.RosCtrlConf.Namespace,
							Name:      "mysecret",
						},
						Data: map[string][]byte{"k1": []byte("v1")},
					},
				),
				data: map[string]string{"k2": "v2"},
			},
			wantData: map[string]string{"k2": "v2"},
		},
		{
			name: "TestSetNotExist",
			args: args{
				name:      "MySecret",
				clientSet: testclient.NewSimpleClientset(),
				data:      map[string]string{"k2": "v2"},
			},
			wantData: map[string]string{"k2": "v2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSecret(tt.args.name, WithClientSet(tt.args.clientSet))
			err := c.SetData(tt.args.data)
			assert.Nil(t, err)

			data, _ := c.GetData()
			assert.Equal(t, tt.wantData, data)
		})
	}
}

func TestSecret_DeleteData(t *testing.T) {
	type args struct {
		name      string
		clientSet *testclient.Clientset
	}
	tests := []struct {
		name     string
		args     args
		wantData map[string]string
	}{
		{
			name: "TestDeleteExist",
			args: args{
				name: "MySecret",
				clientSet: testclient.NewSimpleClientset(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: config.RosCtrlConf.Namespace,
							Name:      "mysecret",
						},
						Data: map[string][]byte{"k1": []byte("v1")},
					},
				),
			},
			wantData: map[string]string{},
		},
		{
			name: "TestSetNotExist",
			args: args{
				name:      "MySecret",
				clientSet: testclient.NewSimpleClientset(),
			},
			wantData: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSecret(tt.args.name, WithClientSet(tt.args.clientSet))
			err := c.DeleteData()
			assert.Nil(t, err)

			data, _ := c.GetData()
			assert.Equal(t, tt.wantData, data)
		})
	}
}
