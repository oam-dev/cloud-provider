package appstack

import (
	"errors"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/golang/mock/gomock"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/appconf"
	roscrd "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/k8s"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/ros"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func newMockAppStackSecret(ctrl *gomock.Controller, data map[string]string) (k8s.SecretInterface, map[string]string) {
	secret := k8s.NewMockSecretInterface(ctrl)
	secret.EXPECT().GetName().Return("MyAppStackSecret").AnyTimes()
	secret.EXPECT().GetData().Return(data, nil).AnyTimes()
	secret.EXPECT().SetData(gomock.Any()).Return(nil).AnyTimes()
	secret.EXPECT().UpdateData(gomock.Any()).Return(nil).AnyTimes()
	return secret, data
}

func newMockProgressingAppStackInfosSecret(ctrl *gomock.Controller, data map[string]string) (k8s.SecretInterface, map[string]string) {
	secret := k8s.NewMockSecretInterface(ctrl)
	secret.EXPECT().GetName().Return("MyProgressingAppStackInfosSecret").AnyTimes()
	secret.EXPECT().GetData().Return(data, nil).AnyTimes()
	secret.EXPECT().SetData(gomock.Any()).
		Do(func(d map[string]string) (err error) {
			data = d
			return nil
		}).
		AnyTimes()
	secret.EXPECT().
		UpdateData(gomock.Any()).
		Do(func(d map[string]string) (err error) {
			data = d
			return nil
		}).
		AnyTimes()
	return secret, data
}

func TestNewAppStack(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "TestNormal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := appconf.NewContext(&appconf.AppConf{}, nil, nil)
			appStack := NewAppStack(ctx)
			assert.NotNil(t, appStack.ctx)
			assert.NotNil(t, appStack.secret)
		})
	}
}

func TestLoadAllProgressingAppStacks(t *testing.T) {
	type args struct {
		AppConf appconf.AppConfInterface
		Data    map[string]string
	}
	tests := []struct {
		name            string
		args            args
		wantRegionId    string
		wantAliUid      string
		wantAppConfName string
	}{
		{
			name: "TestNormal",
			args: args{
				AppConf: &appconf.AppConf{
					ObjectMeta: v1.ObjectMeta{Name: "MyApp"}},
				Data: map[string]string{
					"MyAppSecret": `
						{"AppConfNamespace": "Default",
						"AppConfName": "MyApp",
						"RegionId": "cn-beijing",
						"AliUid": "123456789"}`},
			},
			wantRegionId:    "cn-beijing",
			wantAliUid:      "123456789",
			wantAppConfName: "MyApp",
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, _ := newMockProgressingAppStackInfosSecret(ctrl, tt.args.Data)
			appStacks, _ := LoadProgressingAppStacks(
				nil, nil,
				WithProgressingAppStackInfosSecret(secret),
				WithAppConfGetter(func(
					AppConfNamespace string,
					AppConfName string,
					OamCrdClient *versioned.Clientset,
					RosCrdClient *roscrd.Clientset,
				) (appConf appconf.AppConfInterface, err error) {
					return tt.args.AppConf, nil
				}))

			assert.Len(t, appStacks, 1)
			appStack := appStacks[0]
			assert.Equal(t, tt.wantRegionId, appStack.ctx.RegionId)
			assert.Equal(t, tt.wantAliUid, appStack.ctx.AliUid)
			assert.Equal(t, tt.wantAppConfName, appStack.GetAppName())
		})
	}
}

func TestAppStack_GetSecretName(t *testing.T) {
	type args struct {
		AliUid   string
		RegionId string
		AppName  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestEmptyAliUid",
			args: args{
				AliUid:   "",
				RegionId: "cn-beijing",
				AppName:  "MyApp",
			},
			want: "myapp",
		},
		{
			name: "TestAliUid",
			args: args{
				AliUid:   "123456789",
				RegionId: "cn-beijing",
				AppName:  "MyApp",
			},
			want: "cn-beijing-123456789-myapp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appStack := NewAppStack(
				&appconf.Context{
					AppConf: &appconf.AppConf{
						ObjectMeta: v1.ObjectMeta{Name: tt.args.AppName},
					},
					AliUid:   tt.args.AliUid,
					RegionId: tt.args.RegionId,
				})

			name := appStack.GetSecretName()
			assert.Equal(t, tt.want, name)
		})
	}
}

func TestAppStack_GetOutputSecretName(t *testing.T) {
	type args struct {
		AliUid           string
		RegionId         string
		AppName          string
		CompInstanceName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestEmptyAliUid",
			args: args{
				AliUid:           "",
				RegionId:         "cn-beijing",
				AppName:          "MyApp",
				CompInstanceName: "MyComp",
			},
			want: "myapp-mycomp",
		},
		{
			name: "TestAliUid",
			args: args{
				AliUid:           "123456789",
				RegionId:         "cn-beijing",
				AppName:          "MyApp",
				CompInstanceName: "MyComp",
			},
			want: "cn-beijing-123456789-myapp-mycomp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appStack := NewAppStack(
				&appconf.Context{
					AppConf: &appconf.AppConf{
						ObjectMeta: v1.ObjectMeta{Name: tt.args.AppName},
					},
					AliUid:   tt.args.AliUid,
					RegionId: tt.args.RegionId,
				})

			name := appStack.GetOutputSecretName(tt.args.CompInstanceName)
			assert.Equal(t, tt.want, name)
		})
	}
}

func TestAppStack_GetData(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "TestGetData",
			args: args{data: map[string]string{"k1": "v1"}},
			want: map[string]string{"k1": "v1"},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, _ := newMockAppStackSecret(ctrl, tt.args.data)

			appStack := NewAppStack(
				&appconf.Context{AppConf: &appconf.AppConf{}},
				WithSecret(secret),
			)

			data, _ := appStack.GetData()
			assert.Equal(t, tt.want, data)
		})
	}
}

func TestAppStack_GetStack(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name          string
		args          args
		wantStackId   string
		wantStackName string
	}{
		{
			name: "TestGetStack",
			args: args{
				data: map[string]string{
					ros.StackId:   "abcdefgh-1234-1234-1234-abcdefghijkl",
					ros.StackName: "MyStack",
				},
			},
			wantStackId:   "abcdefgh-1234-1234-1234-abcdefghijkl",
			wantStackName: "MyStack",
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, _ := newMockAppStackSecret(ctrl, tt.args.data)
			appStack := NewAppStack(
				&appconf.Context{AppConf: &appconf.AppConf{}},
				WithSecret(secret),
			)

			stack, _ := appStack.GetStack()
			assert.Equal(t, tt.wantStackId, stack.Id)
			assert.Equal(t, tt.wantStackName, stack.Name)
		})
	}
}

func TestAppStack_GetStatus(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "TestStatusProgressing",
			args: args{
				data: map[string]string{AppStackStatus: Progressing},
			},
			want: Progressing,
		},
		{
			name: "TestStatusReady",
			args: args{
				data: map[string]string{AppStackStatus: Ready},
			},
			want: Ready,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, _ := newMockAppStackSecret(ctrl, tt.args.data)

			appStack := NewAppStack(
				&appconf.Context{AppConf: &appconf.AppConf{}},
				WithSecret(secret),
			)

			status, _ := appStack.GetStatus()
			assert.Equal(t, tt.want, status)
		})
	}
}

func TestAppStack_SetIdAndTemplate(t *testing.T) {
	type args struct {
		data         map[string]string
		StackId      string
		TemplateBody string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "TestSet",
			args: args{
				data:         map[string]string{},
				StackId:      "abcdefgh-1234-1234-1234-abcdefghijkl",
				TemplateBody: "mybody",
			},
			want: map[string]string{
				"StackId":      "abcdefgh-1234-1234-1234-abcdefghijkl",
				"TemplateBody": "mybody",
			},
		},
		{
			name: "TestOverride",
			args: args{
				data: map[string]string{
					"StackId":      "origin",
					"TemplateBody": "origin",
				},
				StackId:      "abcdefgh-1234-1234-1234-abcdefghijkl",
				TemplateBody: "mybody",
			},
			want: map[string]string{
				"StackId":      "abcdefgh-1234-1234-1234-abcdefghijkl",
				"TemplateBody": "mybody",
			},
		},
		{
			name: "TestMerge",
			args: args{
				data: map[string]string{
					"StackId":      "origin",
					"TemplateBody": "mybody",
					"OtherField":   "other",
				},
				StackId:      "abcdefgh-1234-1234-1234-abcdefghijkl",
				TemplateBody: "mybody",
			},
			want: map[string]string{
				"StackId":      "abcdefgh-1234-1234-1234-abcdefghijkl",
				"TemplateBody": "mybody",
				"OtherField":   "other",
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, _ := newMockAppStackSecret(ctrl, tt.args.data)

			appStack := NewAppStack(
				&appconf.Context{AppConf: &appconf.AppConf{}},
				WithSecret(secret),
			)

			err := appStack.SetIdAndTemplate(tt.args.StackId, tt.args.TemplateBody)
			assert.Nil(t, err)
			assert.Equal(t, tt.want, tt.args.data)
		})
	}
}

func TestAppStack_SetError(t *testing.T) {
	type args struct {
		error     error
		updateApp bool
	}
	tests := []struct {
		name        string
		args        args
		wantPhase   v1alpha1.ApplicationPhase
		wantType    v1alpha1.ApplicationConditionType
		wantMessage string
	}{
		{
			name: "TestSdkClientError",
			args: args{
				error:     sdkerrors.NewClientError("mycode", "sdk client error", nil),
				updateApp: true,
			},
			wantPhase:   v1alpha1.ApplicationFailed,
			wantType:    v1alpha1.Error,
			wantMessage: "sdk client error",
		},
		{
			name: "TestSdkClientError_WithoutUpdateApp",
			args: args{
				error:     sdkerrors.NewClientError("mycode", "sdk client error", nil),
				updateApp: false,
			},
			wantPhase:   "",
			wantType:    "",
			wantMessage: "sdk client error",
		},
		{
			name: "TestSdkServerError",
			args: args{
				error:     sdkerrors.NewServerError(400, "sdk server error", ""),
				updateApp: true,
			},
			wantPhase:   v1alpha1.ApplicationFailed,
			wantType:    v1alpha1.Error,
			wantMessage: "sdk server error",
		},
		{
			name: "TestSdkServerError_WithoutUpdateApp",
			args: args{
				error:     sdkerrors.NewServerError(400, "sdk server error", ""),
				updateApp: false,
			},
			wantPhase:   "",
			wantType:    "",
			wantMessage: "sdk server error",
		},
		{
			name: "TestError",
			args: args{
				error:     errors.New("basic error"),
				updateApp: true,
			},
			wantPhase:   v1alpha1.ApplicationFailed,
			wantType:    v1alpha1.Error,
			wantMessage: "basic error",
		},
		{
			name: "TestError_WithoutUpdateApp",
			args: args{
				error:     errors.New("basic error"),
				updateApp: false,
			},
			wantPhase:   "",
			wantType:    "",
			wantMessage: "basic error",
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var appConfPhase v1alpha1.ApplicationPhase
			var appConfType v1alpha1.ApplicationConditionType
			var appConfMessage string

			config.RosCtrlConf.UpdateApp = tt.args.updateApp
			appConf := appconf.NewMockConfigurationInterface(ctrl)
			if tt.args.updateApp {
				appConf.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(c *appconf.Context, phase v1alpha1.ApplicationPhase, type_ v1alpha1.ApplicationConditionType, message string) {
						appConfPhase = phase
						appConfType = type_
						appConfMessage = message
					})
			}

			// app stack secret, and progressing app stack infos secret
			secret, data := newMockAppStackSecret(ctrl, map[string]string{})
			pSecret, _ := newMockProgressingAppStackInfosSecret(ctrl, map[string]string{})

			appStack := NewAppStack(
				&appconf.Context{AppConf: appConf},
				WithSecret(secret),
				WithAppConfFromContextGetter(func(c *appconf.Context) (appconf.AppConfInterface, error) {
					return appConf, nil
				}),
				WithProgressingAppStackInfosSecret(pSecret),
			)

			err := appStack.SetError(tt.args.error)
			assert.Nil(t, err)
			assert.Equal(t, Failed, data[AppStackStatus])
			assert.Equal(t, tt.wantMessage, data[Message])
			assert.Equal(t, tt.wantType, appConfType)
			assert.Equal(t, tt.wantPhase, appConfPhase)
			if appConfPhase != "" {
				assert.Equal(t, tt.wantMessage, appConfMessage)
			} else {
				assert.Equal(t, "", appConfMessage)
			}
		})
	}
}

func TestAppStack_SetProgressing(t *testing.T) {
	type args struct {
		originStatus string
		updateApp    bool
	}
	tests := []struct {
		name       string
		args       args
		wantStatus string
		wantPhase  v1alpha1.ApplicationPhase
	}{
		{
			name: "TestOriginProgressing",
			args: args{
				originStatus: Progressing,
				updateApp:    true,
			},
			wantStatus: Progressing,
			wantPhase:  v1alpha1.ApplicationProgressing,
		},
		{
			name: "TestOriginProgressing_WithoutUpdateApp",
			args: args{
				originStatus: Progressing,
				updateApp:    false,
			},
			wantStatus: Progressing,
			wantPhase:  "",
		},
		{
			name: "TestOriginReady",
			args: args{
				originStatus: Ready,
				updateApp:    true,
			},
			wantStatus: Progressing,
			wantPhase:  v1alpha1.ApplicationProgressing,
		},
		{
			name: "TestOriginReady_WithoutUpdateApp",
			args: args{
				originStatus: Ready,
				updateApp:    false,
			},
			wantStatus: Progressing,
			wantPhase:  "",
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var appConfPhase v1alpha1.ApplicationPhase

			config.RosCtrlConf.UpdateApp = tt.args.updateApp
			appConf := appconf.NewMockConfigurationInterface(ctrl)
			appConf.EXPECT().GetNamespace().Return("DefaultNamespace").AnyTimes()
			appConf.EXPECT().GetName().Return("MyAppConf").AnyTimes()
			if tt.args.updateApp {
				appConf.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(c *appconf.Context, phase v1alpha1.ApplicationPhase, type_ v1alpha1.ApplicationConditionType, message string) {
						appConfPhase = phase
					})
			}

			// app stack secret, and progressing app stack infos secret
			secret, data := newMockAppStackSecret(ctrl, map[string]string{})
			pSecret, _ := newMockProgressingAppStackInfosSecret(ctrl, map[string]string{})

			appStack := NewAppStack(
				&appconf.Context{AppConf: appConf},
				WithSecret(secret),
				WithAppConfFromContextGetter(func(c *appconf.Context) (appconf.AppConfInterface, error) {
					return appConf, nil
				}),
				WithProgressingAppStackInfosSecret(pSecret),
			)

			err := appStack.SetProgressing()
			assert.Nil(t, err)
			assert.Equal(t, tt.wantStatus, data[AppStackStatus])
			assert.Equal(t, tt.wantPhase, appConfPhase)
		})
	}
}

func TestAppStack_SetReady(t *testing.T) {
	type args struct {
		originStatus string
		updateApp    bool
	}
	tests := []struct {
		name       string
		args       args
		wantStatus string
		wantPhase  v1alpha1.ApplicationPhase
	}{
		{
			name: "TestOriginProgressing",
			args: args{
				originStatus: Progressing,
				updateApp:    true,
			},
			wantStatus: Ready,
			wantPhase:  v1alpha1.ApplicationReady,
		},
		{
			name: "TestOriginProgressing_WithoutUpdateApp",
			args: args{
				originStatus: Progressing,
				updateApp:    false,
			},
			wantStatus: Ready,
			wantPhase:  "",
		},
		{
			name: "TestOriginReady",
			args: args{
				originStatus: Ready,
				updateApp:    true,
			},
			wantStatus: Ready,
			wantPhase:  v1alpha1.ApplicationReady,
		},
		{
			name: "TestOriginReady_WithoutUpdateApp",
			args: args{
				originStatus: Ready,
				updateApp:    false,
			},
			wantStatus: Ready,
			wantPhase:  "",
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var appConfPhase v1alpha1.ApplicationPhase

			config.RosCtrlConf.UpdateApp = tt.args.updateApp
			appConf := appconf.NewMockConfigurationInterface(ctrl)
			if tt.args.updateApp {
				appConf.EXPECT().
					UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Do(func(c *appconf.Context, phase v1alpha1.ApplicationPhase, type_ v1alpha1.ApplicationConditionType, message string) {
						appConfPhase = phase
					})
			}

			// app stack secret, and progressing app stack infos secret
			secret, data := newMockAppStackSecret(ctrl, map[string]string{})
			pSecret, _ := newMockProgressingAppStackInfosSecret(ctrl, map[string]string{})

			appStack := NewAppStack(
				&appconf.Context{AppConf: appConf},
				WithSecret(secret),
				WithAppConfFromContextGetter(func(c *appconf.Context) (appconf.AppConfInterface, error) {
					return appConf, nil
				}),
				WithProgressingAppStackInfosSecret(pSecret),
			)

			err := appStack.SetReady()
			assert.Nil(t, err)
			assert.Equal(t, tt.wantStatus, data[AppStackStatus])
			assert.Equal(t, tt.wantPhase, appConfPhase)
		})
	}
}

func TestAppStack_SaveOutputs(t *testing.T) {
	type args struct {
		outputs []map[string]interface{}
	}
	tests := []struct {
		name                string
		args                args
		wantCompSecretsData map[string]map[string]string
	}{
		{
			name: "TestOutputsFromDifferentComps",
			args: args{
				outputs: []map[string]interface{}{
					{ros.OutputKey: "c1.r1", ros.OutputValue: "v1"},
					{ros.OutputKey: "c1.r2", ros.OutputValue: "v2"},
					{ros.OutputKey: "c2.r1", ros.OutputValue: "v1"},
				},
			},
			wantCompSecretsData: map[string]map[string]string{
				"myapp-c1": {"r1": "v1", "r2": "v2"},
				"myapp-c2": {"r1": "v1"},
			},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appSecretData := map[string]string{}
			compSecretsData := map[string]map[string]string{}

			ctx, _ := appconf.NewContext(
				&appconf.AppConf{ObjectMeta: v1.ObjectMeta{Name: "myapp"}},
				nil, nil)
			appSecret := k8s.NewMockSecretInterface(ctrl)
			appSecret.EXPECT().GetName().Return("MyAppStackSecret").AnyTimes()
			appSecret.EXPECT().UpdateData(gomock.Any()).
				Do(func(d map[string]string) error {
					appSecretData = d
					return nil
				})

			appStack := NewAppStack(
				ctx,
				WithSecret(appSecret),
				WithSecretFactory(func(name string, opts ...k8s.SecretOption) k8s.SecretInterface {
					secret := k8s.NewMockSecretInterface(ctrl)
					secret.EXPECT().SetData(gomock.Any()).
						Do(func(d map[string]string) error {
							compSecretsData[name] = d
							return nil
						})
					return secret
				}))
			stack := &ros.Stack{Outputs: tt.args.outputs}

			appStack.SaveOutputs(stack)
			assert.Equal(t, tt.wantCompSecretsData, compSecretsData)
			assert.NotEmpty(t, appSecretData)
		})
	}
}

func TestAppStack_IsProgressing(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestProgressing",
			args: args{
				data: map[string]string{AppStackStatus: Progressing},
			},
			want: true,
		},
		{
			name: "TestNotProgressing1",
			args: args{
				data: map[string]string{AppStackStatus: Ready},
			},
			want: false,
		},
		{
			name: "TestNotProgressing2",
			args: args{
				data: map[string]string{AppStackStatus: Failed},
			},
			want: false,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, _ := newMockAppStackSecret(ctrl, tt.args.data)
			appStack := NewAppStack(
				&appconf.Context{AppConf: &appconf.AppConf{}},
				WithSecret(secret),
			)

			isProgressing, _ := appStack.IsProgressing()
			assert.Equal(t, tt.want, isProgressing)
		})
	}
}

func TestAppStack_IsFailed(t *testing.T) {
	type args struct {
		data map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "TestNotFailed1",
			args: args{
				data: map[string]string{AppStackStatus: Progressing},
			},
			want: false,
		},
		{
			name: "TestNotFailed2",
			args: args{
				data: map[string]string{AppStackStatus: Ready},
			},
			want: false,
		},
		{
			name: "TestFailed",
			args: args{
				data: map[string]string{AppStackStatus: Failed},
			},
			want: true,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, _ := newMockAppStackSecret(ctrl, tt.args.data)
			appStack := NewAppStack(
				&appconf.Context{AppConf: &appconf.AppConf{}},
				WithSecret(secret),
			)

			isFailed, _ := appStack.IsFailed()
			assert.Equal(t, tt.want, isFailed)
		})
	}
}
