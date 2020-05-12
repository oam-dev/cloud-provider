//go:generate mockgen -destination mock_appstack.go -package appstack -source appstack.go
package appstack

import (
	"encoding/json"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/appconf"
	roscrd "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/k8s"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/ros"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	"strings"
	"time"
)

const (
	AppStackName              = "AppStackName"
	AppStackStatus            = "AppStackStatus"
	AppStackOutputSecretNames = "AppStackOutputSecretNames"
	ProgressingAppStackInfos  = "ProgressingAppStackInfos"
	Message                   = "Message"
	TemplateBody              = "TemplateBody"

	Init        = "Init"
	Progressing = "Progressing"
	Ready       = "Ready"
	Failed      = "Failed"
	Deleted     = "Deleted"
)

// globalProgressingAppStackInfosSecret stores all progressing app stack secret names
var globalProgressingAppStackInfosSecret k8s.SecretInterface

func getGlobalProgressingAppStackInfosSecret() k8s.SecretInterface {
	if globalProgressingAppStackInfosSecret == nil {
		globalProgressingAppStackInfosSecret = k8s.NewSecret(ProgressingAppStackInfos)
	}
	return globalProgressingAppStackInfosSecret
}

// appStackOption defines app stack option
type appStackOption struct {
	Secret                         k8s.SecretInterface
	ProgressingAppStackInfosSecret k8s.SecretInterface
	GetAppConfFromContext          appconf.AppConfFromContextGetterFunc
	GetAppConf                     appconf.AppConfGetterFunc
	NewSecret                      func(name string, opts ...k8s.SecretOption) k8s.SecretInterface
}

// AppStackOption has methods to work with app stack option.
type AppStackOption interface {
	apply(*appStackOption)
}

// funcOption defines function used for app stack option
type funcOption struct {
	f func(*appStackOption)
}

// apply executes funcOption's func
func (fdo *funcOption) apply(do *appStackOption) {
	fdo.f(do)
}

// newFuncOption returns function option
func newFuncOption(f func(*appStackOption)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// WithSecret sets secret in app stack option
func WithSecret(secret k8s.SecretInterface) AppStackOption {
	return newFuncOption(func(o *appStackOption) {
		o.Secret = secret
	})
}

// WithAppConfFromContextGetter sets GetAppConfFromContext in app stack option
func WithAppConfFromContextGetter(getAppConfFromContext appconf.AppConfFromContextGetterFunc) AppStackOption {
	return newFuncOption(func(o *appStackOption) {
		o.GetAppConfFromContext = getAppConfFromContext
	})
}

// WithAppConfGetter sets GetAppConfin app stack option
func WithAppConfGetter(getAppConf appconf.AppConfGetterFunc) AppStackOption {
	return newFuncOption(func(o *appStackOption) {
		o.GetAppConf = getAppConf
	})
}

// WithSecretFactory sets NetSecret in app stack option
func WithSecretFactory(newSecret func(name string, opts ...k8s.SecretOption) k8s.SecretInterface) AppStackOption {
	return newFuncOption(func(o *appStackOption) {
		o.NewSecret = newSecret
	})
}

// WithProgressingAppStackInfosSecret sets ProgressingAppStackInfosSecret in app stack option
func WithProgressingAppStackInfosSecret(progressingAppStacksSecret k8s.SecretInterface) AppStackOption {
	return newFuncOption(func(o *appStackOption) {
		o.ProgressingAppStackInfosSecret = progressingAppStacksSecret
	})
}

// AppStackInterface has methods to work with app stack resources.
type AppStackInterface interface {
	GetName() string
	GetAppName() string
	GetSecretName() string
	GetOutputSecretName(compInstanceName string) string
	GetData() (data map[string]string, err error)
	GetStack() (stack *ros.Stack, err error)
	GetStatus() (value string, err error)
	GetContext() (ctx *appconf.Context)
	SetIdAndTemplate(stackId string, templateBody string) (err error)
	SetError(e error) (err error)
	SetProgressing() (err error)
	SetReady() (err error)
	SaveOutputs(stack *ros.Stack)
	IsProgressing() (progressing bool, err error)
	IsFailed() (failed bool, err error)
	Delete() (err error)
	WaitUntilDone() (status string)
}

// AppStackInfo used for updating processing app stacks
type AppStackInfo struct {
	AppConfNamespace string `json:"AppConfNamespace,omitempty"`
	AppConfName      string `json:"AppConfName,omitempty"`
	RegionId         string `json:"RegionId,omitempty"`
	AliUid           string `json:"AliUid,omitempty"`
}

// AppStack implements AppStackInterface
type AppStack struct {
	name                       string
	ctx                        *appconf.Context
	secret                     k8s.SecretInterface
	progressingAppStacksSecret k8s.SecretInterface
	getAppConfFromContext      appconf.AppConfFromContextGetterFunc
	getAppConf                 appconf.AppConfGetterFunc
	newSecret                  func(name string, opts ...k8s.SecretOption) k8s.SecretInterface
	status                     string
}

// NewAppStack returns an app stack
func NewAppStack(ctx *appconf.Context, opts ...AppStackOption) *AppStack {
	// init opts
	o := &appStackOption{}
	for _, opt := range opts {
		opt.apply(o)
	}

	appStack := &AppStack{
		ctx:                        ctx,
		secret:                     o.Secret,
		progressingAppStacksSecret: o.ProgressingAppStackInfosSecret,
		getAppConfFromContext:      o.GetAppConfFromContext,
		getAppConf:                 o.GetAppConf,
		newSecret:                  o.NewSecret,
		status:                     Init,
	}

	if appStack.secret == nil {
		secretName := appStack.GetSecretName()
		appStack.secret = k8s.NewSecret(secretName)
		appStack.name = secretName
	} else {
		appStack.name = appStack.secret.GetName()
	}

	if appStack.progressingAppStacksSecret == nil {
		appStack.progressingAppStacksSecret = getGlobalProgressingAppStackInfosSecret()
	}

	if appStack.getAppConfFromContext == nil {
		appStack.getAppConfFromContext = appconf.GetAppConfFromContext
	}

	if appStack.getAppConf == nil {
		appStack.getAppConf = appconf.GetAppConf
	}

	if appStack.newSecret == nil {
		appStack.newSecret = k8s.NewSecret
	}

	return appStack
}

// LoadProgressingAppStacks returns all app stacks in Progressing and an error if there is any.
func LoadProgressingAppStacks(
	OamCrdClient *versioned.Clientset,
	RosCrdClient *roscrd.Clientset,
	opts ...AppStackOption) (appStacks []*AppStack, err error) {
	// init opts
	o := &appStackOption{}
	for _, opt := range opts {
		opt.apply(o)
	}

	getAppConf := o.GetAppConf
	if getAppConf == nil {
		getAppConf = appconf.GetAppConf
	}

	progressingAppStackInfosSecret := o.ProgressingAppStackInfosSecret
	if progressingAppStackInfosSecret == nil {
		progressingAppStackInfosSecret = getGlobalProgressingAppStackInfosSecret()
	}

	data, err := progressingAppStackInfosSecret.GetData()
	if err != nil {
		return
	}

	for appStackSecretName, appStackData := range data {
		var appStackInfo = &AppStackInfo{}
		err = json.Unmarshal([]byte(appStackData), appStackInfo)
		if err != nil {
			logging.Default.Error(err, "AppStackData is invalid",
				"AppStackSecretName", appStackSecretName,
				"AppStackData", appStackData)
			return
		}

		appConf, err := getAppConf(
			appStackInfo.AppConfNamespace,
			appStackInfo.AppConfName,
			OamCrdClient,
			RosCrdClient)
		if err != nil {
			return appStacks, err
		}

		ctx, err := appconf.NewContext(appConf, OamCrdClient, RosCrdClient)
		if err != nil {
			return appStacks, err
		}
		ctx.AliUid = appStackInfo.AliUid
		ctx.RegionId = appStackInfo.RegionId

		appStack := NewAppStack(ctx, opts...)
		appStacks = append(appStacks, appStack)
	}
	return
}

// GetName returns name of app stack.
func (c *AppStack) GetName() string {
	return c.name
}

// GetAppName returns name of app.
func (c *AppStack) GetAppName() string {
	return c.ctx.AppName
}

// GetSecretName returns secret name of app stack.
func (c *AppStack) GetSecretName() string {
	if c.ctx.AliUid == "" {
		return strings.ToLower(c.ctx.AppConf.GetName())
	} else {
		return strings.ToLower(c.ctx.RegionId + "-" + c.ctx.AliUid + "-" + c.ctx.AppConf.GetName())
	}
}

// GetOutputSecretName takes compInstanceName and returns the corresponding secret name.
func (c *AppStack) GetOutputSecretName(compInstanceName string) string {
	if c.ctx.AliUid == "" {
		return strings.ToLower(c.ctx.AppConf.GetName() + "-" + compInstanceName)
	} else {
		return strings.ToLower(c.ctx.RegionId + "-" + c.ctx.AliUid + "-" + c.ctx.AppConf.GetName() + "-" + compInstanceName)
	}
}

// GetData returns the data of the secret, and an error, if there is any.
func (c *AppStack) GetData() (data map[string]string, err error) {
	data, err = c.secret.GetData()
	return
}

// GetStack returns the stack, and an error, if there is any.
func (c *AppStack) GetStack() (stack *ros.Stack, err error) {
	data, err := c.secret.GetData()
	if err != nil {
		return
	}

	if data != nil && data[ros.StackId] != "" {
		stack = &ros.Stack{
			Client: c.ctx.RosClient,
			Id:     data[ros.StackId],
			Name:   data[ros.StackName],
		}
	}
	return
}

// GetStatus returns the AppStackStatus of app stack, and an error, if there is any.
func (c *AppStack) GetStatus() (value string, err error) {
	if c.status == Init || c.status == "" {
		value, err = c.get(AppStackStatus)
		if value == "" && err == nil {
			c.status = Deleted
		}
		return
	}
	return c.status, nil
}

// GetStatus returns the AppStackStatus of app stack, and an error, if there is any.
func (c *AppStack) GetContext() (ctx *appconf.Context) {
	return c.ctx
}

// SetIdAndTemplate set stack ID and template body. Returns an error if one occurs.
func (c *AppStack) SetIdAndTemplate(stackId string, templateBody string) (err error) {
	err = c.set(ros.StackId, stackId, TemplateBody, templateBody)
	return
}

// SetError set AppStackStatus to Failed and error message. Returns an error if one occurs.
func (c *AppStack) SetError(e error) (err error) {
	logging.Default.Info("Set error msg to app stack", "error", e)

	var message string
	switch e.(type) {
	case sdkerrors.Error:
		ee := e.(sdkerrors.Error)
		message = ee.Message()
	default:
		message = e.Error()
	}

	c.status = Failed
	err = c.set(AppStackStatus, Failed, Message, message)
	if err != nil {
		return
	}

	err = c.removeProgressingAppStackInfoToSecret()
	if err != nil {
		return
	}

	err = c.maybeSetAppCondition(v1alpha1.ApplicationFailed, message)
	return
}

// SetProgressing set AppStackStatus to Progressing. Returns an error if one occurs.
func (c *AppStack) SetProgressing() (err error) {
	c.status = Progressing
	err = c.set(AppStackStatus, Progressing)
	if err != nil {
		return
	}

	err = c.addProgressingAppStackInfoToSecret()
	if err != nil {
		return
	}

	err = c.maybeSetAppCondition(v1alpha1.ApplicationProgressing, "")
	return
}

// SetReady set AppStackStatus to Ready. Returns an error if one occurs.
func (c *AppStack) SetReady() (err error) {
	c.status = Ready
	err = c.set(AppStackStatus, Ready, Message, "")
	if err != nil {
		return
	}

	err = c.removeProgressingAppStackInfoToSecret()
	if err != nil {
		return
	}

	err = c.maybeSetAppCondition(v1alpha1.ApplicationReady, "")
	return
}

// SaveOutputs saves outputs of stack.
func (c *AppStack) SaveOutputs(stack *ros.Stack) {
	data := make(map[string]map[string]string)

	for _, output := range stack.Outputs {
		// handle key
		outputKey := output[ros.OutputKey].(string)
		keySlice := strings.Split(outputKey, ".")
		if len(keySlice) < 2 {
			logging.Default.Info("Unexpected OutputKey", ros.OutputKey, outputKey)
			continue
		}
		compInstanceName := keySlice[0]
		key := strings.Join(keySlice[1:], ".")

		// handle value
		var value string
		outputValue := output[ros.OutputValue]
		switch outputValue.(type) {
		case string:
			value = outputValue.(string)
		default:
			byteValue, err := json.Marshal(value)
			if err != nil {
				logging.Default.Error(err, "Unexpected OutputValue", ros.OutputValue, outputValue)
				continue
			}
			value = string(byteValue)
		}

		secretName := c.GetOutputSecretName(compInstanceName)
		secretData, ok := data[secretName]
		if ok {
			secretData[key] = value
		} else {
			data[secretName] = map[string]string{key: value}
		}
	}

	// save outputs to several secrets
	var secretNames []string
	for secretName, secretData := range data {
		secret := c.newSecret(secretName)
		err := secret.SetData(secretData)
		if err != nil {
			logging.Default.Error(err, "Save output to secret error", "AppStackName", secretName)
			continue
		}
		secretNames = append(secretNames, secretName)
	}

	// save outputs secret names to app stack secret
	appStackOutputSecretNames := strings.Join(secretNames, ",")
	secretData := map[string]string{AppStackOutputSecretNames: appStackOutputSecretNames}
	err := c.secret.UpdateData(secretData)
	if err != nil {
		logging.Default.Error(err, "Save output secret names to app stack secret error",
			"AppStackName", c.secret.GetName(),
			"AppStackOutputSecretNames", appStackOutputSecretNames,
		)
	}
}

// IsProgressing returns whether app stack is Progressing, and an error, if there is any.
func (c *AppStack) IsProgressing() (progressing bool, err error) {
	status, err := c.GetStatus()
	if err != nil {
		return false, err
	}
	progressing = status == Progressing
	return
}

// IsFailed returns whether app stack is Failed, and an error, if there is any.
func (c *AppStack) IsFailed() (failed bool, err error) {
	status, err := c.GetStatus()
	if err != nil {
		return false, err
	}
	failed = status == Failed
	return
}

// Delete deletes outputs and data. Returns an error if one occurs.
func (c *AppStack) Delete() (err error) {
	// delete outputs
	data, err := c.secret.GetData()
	if err != nil {
		return
	}
	appStackOutputSecretNames := data[AppStackOutputSecretNames]
	secretNames := strings.Split(appStackOutputSecretNames, ",")
	for _, name := range secretNames {
		if name == "" {
			continue
		}
		secret := k8s.NewSecret(name)
		err = secret.DeleteData()
		if err != nil {
			return
		}
	}

	// delete from k8s secret
	err = c.secret.DeleteData()
	if err != nil {
		return
	}

	err = c.removeProgressingAppStackInfoToSecret()
	c.status = Deleted
	return
}

// WaitUntilDone waits app stack until done.
func (c *AppStack) WaitUntilDone() (status string) {
	for {
		time.Sleep(5 * time.Second)
		logging.Default.Info("Waiting app done", "AppStackName", c.ctx.AppName)
		status, err := c.GetStatus()
		if err != nil {
			logging.Default.Error(err, "Waiting app done error")
		}
		if status == Ready || status == Failed || status == Deleted || status == "" {
			logging.Default.Info("App done", "AppStackName", c.ctx.AppName, "AppStatus", status)
			return status
		}
	}
}

// get data from k8s secret
func (c *AppStack) get(key string) (value string, err error) {
	data, err := c.secret.GetData()
	if err != nil {
		return
	}
	value = data[key]
	return
}

// get and update data from k8s secret
func (c *AppStack) set(keysAndValues ...string) (err error) {
	data, err := c.secret.GetData()
	if err != nil {
		return
	}
	for i := 0; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		data[key] = value
	}
	err = c.secret.UpdateData(data)
	return
}

// addProgressingAppStackInfoToSecret adds progressing app stack info to ProgressingAppStackInfos secret
func (c *AppStack) addProgressingAppStackInfoToSecret() (err error) {
	appStackInfo := AppStackInfo{
		AppConfNamespace: c.ctx.AppConf.GetNamespace(),
		AppConfName:      c.ctx.AppConf.GetName(),
		RegionId:         c.ctx.RegionId,
		AliUid:           c.ctx.AliUid,
	}
	appStackData, err := json.Marshal(appStackInfo)
	if err != nil {
		return
	}

	appStackSecretName := c.secret.GetName()
	err = c.progressingAppStacksSecret.UpdateData(map[string]string{
		appStackSecretName: string(appStackData),
	})
	return
}

// removeProgressingAppStackInfoToSecret removes progressing app stack info to ProgressingAppStackInfos secret
func (c *AppStack) removeProgressingAppStackInfoToSecret() (err error) {
	data, err := c.progressingAppStacksSecret.GetData()
	if err != nil {
		return
	}
	appStackSecretName := c.secret.GetName()
	delete(data, appStackSecretName)

	err = c.progressingAppStacksSecret.SetData(data)
	return
}

func (c *AppStack) maybeSetAppCondition(phase v1alpha1.ApplicationPhase, message string) (err error) {
	if !config.RosCtrlConf.UpdateApp {
		return
	}

	var type_ v1alpha1.ApplicationConditionType
	if phase == v1alpha1.ApplicationFailed {
		type_ = v1alpha1.Error
	} else {
		type_ = v1alpha1.Ready
	}

	updateConf, err := c.getAppConfFromContext(c.ctx)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		logging.Default.Error(err, "Get app conf error while set app condition")
		return err
	}

	err = updateConf.UpdateStatus(c.ctx, phase, type_, message)
	if err != nil {
		logging.Default.Error(err, "Update app conf error")
	}
	return
}
