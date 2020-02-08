package application

import (
	"encoding/json"
	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/k8s"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/ros"
	rosv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"strings"
)

const (
	AppName                   = "AppName"
	AppStackStatus            = "AppStackStatus"
	AppStackOutputSecretNames = "AppStackOutputSecretNames"
	Message                   = "Message"
	TemplateBody              = "TemplateBody"

	Progressing = "Progressing"
	Ready       = "Ready"
	Failed      = "Failed"
)

type SetAppStackStatusFunc func(*rosv1alpha1.ApplicationConfiguration, string) error

func IsAppStackProgressing(data map[string]string) bool {
	return data[AppStackStatus] == Progressing
}

func IsAppStackFailed(data map[string]string) bool {
	return data[AppStackStatus] == Failed
}

func GetAppStackSecretName(rosContext *ros.Context) string {
	if rosContext.AliUid == "" {
		return strings.ToLower(rosContext.AppConf.Name)
	} else {
		return strings.ToLower(rosContext.RegionId + "-" + rosContext.AliUid + "-" + rosContext.AppConf.Name)
	}
}

func GetAppStackOutputSecretName(rosContext *ros.Context, compInstanceName string) string {
	if rosContext.AliUid == "" {
		return strings.ToLower(rosContext.AppConf.Name + "-" + compInstanceName)
	} else {
		return strings.ToLower(rosContext.RegionId + "-" + rosContext.AliUid + "-" + rosContext.AppConf.Name + "-" + compInstanceName)
	}
}

func GetAppStackData(rosContext *ros.Context) (data map[string]string, err error) {
	secretName := GetAppStackSecretName(rosContext)
	data, err = k8s.GetSecretData(secretName)
	return
}

func GetStackFromAppStack(rosContext *ros.Context) (stack *ros.Stack, err error) {
	secretName := GetAppStackSecretName(rosContext)
	data, err := k8s.GetSecretData(secretName)
	if err != nil {
		return
	}

	if data != nil && data[ros.StackId] != "" {
		stack = &ros.Stack{
			Id:   data[ros.StackId],
			Name: data[ros.StackName],
		}
	}
	return
}

func GetAppStackStatus(rosContext *ros.Context) (value string, err error) {
	value, err = getAppStack(rosContext, AppStackStatus)
	return
}

func SetAppStackIdAndTemplate(rosContext *ros.Context, stackId string, templateBody string) (err error) {
	err = setAppStack(rosContext, ros.StackId, stackId, TemplateBody, templateBody)
	return
}

func SetAppStackError(rosContext *ros.Context, error error) (err error) {
	logging.Default.Info("Set error msg to app stack", "error", error)

	var message string
	switch error.(type) {
	case sdkerrors.Error:
		err := error.(sdkerrors.Error)
		message = err.Message()
	default:
		message = error.Error()
	}

	err = setAppStack(rosContext, AppStackStatus, Failed, Message, message)
	if err != nil {
		return
	}

	err = maybeSetAppCondition(rosContext, v1alpha1.ApplicationFailed, message)
	return
}

func SetAppStackProgressing(rosContext *ros.Context) (err error) {
	err = setAppStack(rosContext, AppStackStatus, Progressing)
	if err != nil {
		return
	}

	err = maybeSetAppCondition(rosContext, v1alpha1.ApplicationProgressing, "")
	return
}

func SetAppStackReady(rosContext *ros.Context) (err error) {
	err = setAppStack(rosContext, AppStackStatus, Ready, Message, "")
	if err != nil {
		return
	}

	err = maybeSetAppCondition(rosContext, v1alpha1.ApplicationReady, "")
	return
}

func getAppStack(rosContext *ros.Context, key string) (value string, err error) {
	// get data from k8s secret
	secretName := GetAppStackSecretName(rosContext)
	data, err := k8s.GetSecretData(secretName)
	if err != nil {
		return
	}
	value = data[key]
	return
}

func setAppStack(rosContext *ros.Context, keysAndValues ...string) (err error) {
	// get and update data from k8s secret
	secretName := GetAppStackSecretName(rosContext)
	data, err := k8s.GetSecretData(secretName)
	if err != nil {
		return
	}
	for i := 0; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		data[key] = value
	}
	err = k8s.UpdateSecretData(secretName, data)
	return
}

func maybeSetAppCondition(rosContext *ros.Context, phase v1alpha1.ApplicationPhase, message string) (err error) {
	if !config.RosCtrlConf.UpdateApp {
		return
	}

	var type_ v1alpha1.ApplicationConditionType
	if phase == v1alpha1.ApplicationFailed {
		type_ = v1alpha1.Error
	} else {
		type_ = v1alpha1.Ready
	}

	updateConf, err := rosContext.GetAppConf()
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		logging.Default.Error(err, "Get app conf error while set app condition")
		return err
	}

	err = rosContext.UpdateAppConfStatus(updateConf, phase, type_, message)
	if err != nil {
		logging.Default.Error(err, "Update app conf error")
	}
	return
}

func DeleteAppStack(rosContext *ros.Context) (err error) {
	secretName := GetAppStackSecretName(rosContext)

	// delete outputs
	data, err := k8s.GetSecretData(secretName)
	if err != nil {
		return
	}
	appStackOutputSecretNames := data[AppStackOutputSecretNames]
	secretNames := strings.Split(appStackOutputSecretNames, ",")
	for _, name := range secretNames {
		if name == "" {
			continue
		}
		err = k8s.DeleteSecretData(name)
		if err != nil {
			return
		}
	}

	// delete from k8s secret
	err = k8s.DeleteSecretData(secretName)
	return
}

func SaveAppStackOutputs(rosContext *ros.Context, stack *ros.Stack) {
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

		secretName := GetAppStackOutputSecretName(rosContext, compInstanceName)
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
		err := k8s.SetSecretData(secretName, secretData)
		if err != nil {
			logging.Default.Error(err, "Save output to secret error", "SecretName", secretName)
			continue
		}
		secretNames = append(secretNames, secretName)
	}

	// save outputs secret names to app stack secret
	appStackOutputSecretNames := strings.Join(secretNames, ",")
	secretData := map[string]string{AppStackOutputSecretNames: appStackOutputSecretNames}
	secretName := GetAppStackSecretName(rosContext)
	err := k8s.UpdateSecretData(secretName, secretData)
	if err != nil {
		logging.Default.Error(err, "Save output secret names to app stack secret error",
			"SecretName", secretName,
			"AppStackOutputSecretNames", appStackOutputSecretNames,
		)
	}
}
