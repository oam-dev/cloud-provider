package ros

import (
	"encoding/json"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/aliyun"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
)

type Context struct {
	AliUid    string
	RegionId  string
	AppConf   *v1alpha1.ApplicationConfiguration
	K8sClient *versioned.Clientset
	RosClient *rosapi.Client
}

func GetContext(appConf *v1alpha1.ApplicationConfiguration, k8sClient *versioned.Clientset) (context *Context, err error) {
	client := &rosapi.Client{}
	context = &Context{
		AppConf:   appConf,
		K8sClient: k8sClient,
		RosClient: client,
		RegionId:  config.RosCtrlConf.RegionId, // maybe changed by being assigned in scope
	}

	// get from scope
	for _, scope := range appConf.Spec.Scopes {
		if scope.Name == config.RESOURCE_IDENTITY && scope.Type == config.RESOURCE_IDENTITY_TYPE {
			logging.Default.Info("Identity scope detected", "AppConfName", appConf.ObjectMeta.Name)

			resourceIdentity := &aliyun.AliyunResourceIdentity{}
			err = json.Unmarshal(scope.Properties.Raw, resourceIdentity)
			if err != nil {
				return
			}

			// get ak from secret
			secretName := resourceIdentity.IdentityAsKey()
			logging.Default.Info("Get aliyun credential by aliyun resource identity", "Identity", resourceIdentity)
			credential, err := aliyun.ReadCredentialFromSecret(secretName)
			if err != nil {
				return context, err
			}

			// context
			context.AliUid = resourceIdentity.AliUid
			context.RegionId = resourceIdentity.RegionId
			if context.RegionId == "" {
				context.RegionId = config.RosCtrlConf.RegionId
			}

			// init client
			err = initClient(context, client, credential)
			return context, err
		}
	}

	// get from secret
	credentialSecretName := config.RosCtrlConf.CredentialSecretName
	if credentialSecretName != "" {
		logging.Default.Info("Get aliyun credential by credential secret name", "CredentialSecretName", credentialSecretName)
		credential, err := aliyun.ReadCredentialFromSecret(credentialSecretName)
		if err != nil {
			return context, err
		}

		err = initClient(context, client, credential)
	} else { // get from ak
		err = client.InitWithAccessKey(
			context.RegionId,
			config.RosCtrlConf.AccessKeyId,
			config.RosCtrlConf.AccessKeySecret,
		)
	}
	return
}

func initClient(context *Context, client *rosapi.Client, credential *aliyun.AliyunCredential) (err error) {
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
