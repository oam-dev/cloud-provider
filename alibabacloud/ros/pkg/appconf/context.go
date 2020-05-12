package appconf

import (
	"encoding/json"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/aliyun"
	roscrd "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
)

type Context struct {
	AliUid       string
	RegionId     string
	DryRun       bool
	AppName      string
	AppConf      AppConfInterface
	OamCrdClient *versioned.Clientset
	RosCrdClient *roscrd.Clientset
	RosClient    *rosapi.Client
}

func NewContext(
	appConf AppConfInterface,
	oamCrdClient *versioned.Clientset,
	rosCrdClient *roscrd.Clientset) (context *Context, err error) {

	rosClient := &rosapi.Client{}
	context = &Context{
		AppName:      appConf.GetName(),
		AppConf:      appConf,
		OamCrdClient: oamCrdClient,
		RosCrdClient: rosCrdClient,
		RosClient:    rosClient,
		DryRun:       config.RosCtrlConf.DryRun,
		RegionId:     config.RosCtrlConf.RegionId, // maybe changed by being assigned in scope
	}

	initialized, err := initContextFromScope(appConf, context)
	if err != nil {
		return nil, err
	}
	if initialized {
		return context, nil
	}

	err = initContextFromConfig(context)
	if err != nil {
		return nil, err
	}
	return context, nil
}

func initContextFromConfig(context *Context) error {
	// Don't init from Config if dry run
	if context.DryRun {
		return nil
	}

	// get from secret
	credentialSecretName := config.RosCtrlConf.CredentialSecretName
	if credentialSecretName != "" {
		logging.Default.Info("Get aliyun credential by credential secret name", "CredentialSecretName", credentialSecretName)
		credential, err := aliyun.ReadCredentialFromSecretName(credentialSecretName)
		if err != nil {
			return err
		}
		return initRosClient(context, context.RosClient, credential)
	}
	// get from ak
	return context.RosClient.InitWithAccessKey(
		context.RegionId,
		config.RosCtrlConf.AccessKeyId,
		config.RosCtrlConf.AccessKeySecret,
	)
}

func initContextFromScope(appConf AppConfInterface, context *Context) (bool, error) {
	// Don't init from scope if dry run
	if context.DryRun {
		return false, nil
	}

	for _, scope := range appConf.GetScopes() {
		if scope.Name == config.RESOURCE_IDENTITY && scope.Type == config.RESOURCE_IDENTITY_TYPE {
			logging.Default.Info("Identity scope detected", "AppConfName", appConf.GetObjectMeta().Name)

			resourceIdentity := &aliyun.AliyunResourceIdentity{}
			err := json.Unmarshal(scope.Properties.Raw, resourceIdentity)
			if err != nil {
				return false, err
			}

			// get ak from secret
			secretName := resourceIdentity.IdentityAsKey()
			logging.Default.Info("Get aliyun credential by aliyun resource identity", "Identity", resourceIdentity)
			credential, err := aliyun.ReadCredentialFromSecretName(secretName)
			if err != nil {
				return false, err
			}

			// context
			context.AliUid = resourceIdentity.AliUid
			context.RegionId = resourceIdentity.RegionId
			if context.RegionId == "" {
				context.RegionId = config.RosCtrlConf.RegionId
			}

			// init rosClient
			err = initRosClient(context, context.RosClient, credential)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func initRosClient(context *Context, client *rosapi.Client, credential *aliyun.AliyunCredential) (err error) {
	if credential.SecurityToken != "" {
		err = client.InitWithStsToken(
			context.RegionId,
			credential.AccessKeyId,
			credential.AccessKeySecret,
			credential.SecurityToken,
		)
	} else {
		err = client.InitWithAccessKey(
			context.RegionId,
			credential.AccessKeyId,
			credential.AccessKeySecret,
		)
	}
	return
}
