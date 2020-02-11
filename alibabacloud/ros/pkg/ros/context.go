package ros

import (
	"encoding/json"
	"errors"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/aliyun"
	roscrd "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/rosapi"
	rosv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Context struct {
	AliUid       string
	RegionId     string
	AppConf      *rosv1alpha1.ApplicationConfiguration
	OamCrdClient *versioned.Clientset
	RosCrdClient *roscrd.Clientset
	RosClient    *rosapi.Client
}

func (c *Context) GetAppConf() (appConf *rosv1alpha1.ApplicationConfiguration, err error) {
	if c.OamCrdClient != nil {
		appConfInterface := c.OamCrdClient.CoreV1alpha1().ApplicationConfigurations(c.AppConf.Namespace)
		appConf_, err := appConfInterface.Get(c.AppConf.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		appConf, _ = rosv1alpha1.NewApplicationConfiguration(appConf_)
		return appConf, nil
	} else if c.RosCrdClient != nil {
		appConfInterface := c.RosCrdClient.RosV1alpha1().RosStacks(c.AppConf.Namespace)
		appConf_, err := appConfInterface.Get(c.AppConf.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		appConf, _ = rosv1alpha1.NewApplicationConfiguration(appConf_)
		return appConf, nil
	} else {
		return nil, errors.New("no client found")
	}
}

func (c *Context) UpdateAppConf(appConf *rosv1alpha1.ApplicationConfiguration) (err error) {
	if c.OamCrdClient != nil {
		appConfInterface := c.OamCrdClient.CoreV1alpha1().ApplicationConfigurations(appConf.Namespace)
		_, err = appConfInterface.Update(appConf.ToOamApplicationConfiguration())
		if err == nil {
			c.AppConf = appConf
		}
		return
	} else if c.RosCrdClient != nil {
		appConfInterface := c.RosCrdClient.RosV1alpha1().RosStacks(appConf.Namespace)
		_, err = appConfInterface.Update(appConf.ToRosStack())
		if err == nil {
			c.AppConf = appConf
		}
		return
	}
	return errors.New("no client found")
}

func (c *Context) UpdateAppConfStatus(
	appConf *rosv1alpha1.ApplicationConfiguration,
	phase v1alpha1.ApplicationPhase, type_ v1alpha1.ApplicationConditionType,
	message string) (err error) {

	if c.OamCrdClient == nil && c.RosCrdClient == nil {
		return errors.New("no client found")
	}

	updateConf := appConf
	if updateConf.Status.Conditions == nil {
		condition := v1alpha1.ApplicationCondition{
			Type:               type_,
			Status:             corev1.ConditionTrue,
			LastUpdateTime:     v1.Now(),
			LastTransitionTime: v1.Now(),
			Reason:             message,
			Message:            message,
		}

		updateConf.Status = v1alpha1.ApplicationConfigurationStatus{
			Phase:      phase,
			Conditions: []v1alpha1.ApplicationCondition{condition},
		}
	} else {
		updateConf.Status.Phase = phase
		condition := &updateConf.Status.Conditions[0]
		condition.Type = type_
		condition.LastUpdateTime = v1.Now()
		condition.LastTransitionTime = v1.Now()
		if message != "" {
			condition.Message = message
		}
	}

	if c.OamCrdClient != nil {
		appConfInterface := c.OamCrdClient.CoreV1alpha1().ApplicationConfigurations(appConf.Namespace)
		_, err = appConfInterface.UpdateStatus(updateConf.ToOamApplicationConfiguration())
	} else {
		appConfInterface := c.RosCrdClient.RosV1alpha1().RosStacks(appConf.Namespace)
		_, err = appConfInterface.UpdateStatus(updateConf.ToRosStack())
	}

	if err == nil {
		c.AppConf = updateConf
	}

	return
}

func GetContext(
	appConf *rosv1alpha1.ApplicationConfiguration,
	oamCrdClient *versioned.Clientset,
	rosCrdClient *roscrd.Clientset) (context *Context, err error) {

	rosClient := &rosapi.Client{}
	context = &Context{
		AppConf:      appConf,
		OamCrdClient: oamCrdClient,
		RosCrdClient: rosCrdClient,
		RosClient:    rosClient,
		RegionId:     config.RosCtrlConf.RegionId, // maybe changed by being assigned in scope
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

			// init rosClient
			err = initRosClient(context, rosClient, credential)
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

		err = initRosClient(context, rosClient, credential)
	} else { // get from ak
		err = rosClient.InitWithAccessKey(
			context.RegionId,
			config.RosCtrlConf.AccessKeyId,
			config.RosCtrlConf.AccessKeySecret,
		)
	}
	return
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
