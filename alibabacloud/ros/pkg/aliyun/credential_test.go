package aliyun

import (
	"github.com/golang/mock/gomock"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/k8s"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAliyunResourceIdentity_IdentityAsKey(t *testing.T) {
	type fields struct {
		AppName  string
		AliUid   string
		RegionId string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TestNormal",
			fields: fields{
				AppName:  "myapp",
				RegionId: "cn-beijing",
				AliUid:   "123456789",
			},
			want: "myapp.cn-beijing.123456789",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ari := &AliyunResourceIdentity{
				AppName:  tt.fields.AppName,
				AliUid:   tt.fields.AliUid,
				RegionId: tt.fields.RegionId,
			}
			assert.Equal(t, tt.want, ari.IdentityAsKey())
		})
	}
}

func TestReadCredentialFromSecret(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name           string
		args           args
		wantCredential *AliyunCredential
		wantErr        bool
	}{
		{
			name: "TestAK",
			args: args{data: map[string]string{
				"AccessKeyId":     "id",
				"AccessKeySecret": "secret",
			}},
			wantCredential: &AliyunCredential{
				AccessKeyId:     "id",
				AccessKeySecret: "secret",
				SecurityToken:   "",
				Expiration:      "",
			},
			wantErr: false,
		},
		{
			name: "TestStsToken",
			args: args{data: map[string]string{
				"AccessKeyId":     "id",
				"AccessKeySecret": "secret",
				"SecurityToken":   "token",
				"Expiration":      "expiration",
			}},
			wantCredential: &AliyunCredential{
				AccessKeyId:     "id",
				AccessKeySecret: "secret",
				SecurityToken:   "token",
				Expiration:      "expiration",
			},
			wantErr: false,
		},
		{
			name: "TestLessThanTwo",
			args: args{data: map[string]string{
				"AccessKeyId": "id",
			}},
			wantCredential: nil,
			wantErr:        true,
		},
		{
			name: "TestAccessKeyIdEmpty",
			args: args{data: map[string]string{
				"AccessKeyId":     "",
				"AccessKeySecret": "secret",
			}},
			wantCredential: nil,
			wantErr:        true,
		},
		{
			name: "TestAccessKeySecretEmpty",
			args: args{data: map[string]string{
				"AccessKeyId":     "id",
				"AccessKeySecret": "",
			}},
			wantCredential: nil,
			wantErr:        true,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := k8s.NewMockSecretInterface(ctrl)
			secret.EXPECT().GetData().Return(tt.args.data, nil)
			secret.EXPECT().GetName().AnyTimes().Return("mysecret")

			gotCredential, err := ReadCredentialFromSecret(secret)
			assert.Equal(t, tt.wantCredential, gotCredential)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
