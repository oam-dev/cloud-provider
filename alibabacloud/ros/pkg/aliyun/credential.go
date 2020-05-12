package aliyun

import (
	"errors"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/k8s"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
)

type AliyunCredential struct {
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
	Expiration      string
}

type AliyunResourceIdentity struct {
	AppName  string `json:"appName"`
	AliUid   string `json:"aliyunAccountUid"`
	RegionId string `json:"regionId"`
}

func (ari *AliyunResourceIdentity) IdentityAsKey() string {
	return ari.AppName + "." + ari.RegionId + "." + ari.AliUid
}

func ReadCredentialFromSecretName(secretName string) (credential *AliyunCredential, err error) {
	secret := k8s.NewSecret(secretName)
	return ReadCredentialFromSecret(secret)
}

func ReadCredentialFromSecret(secret k8s.SecretInterface) (credential *AliyunCredential, err error) {
	credentialSecretData, err := secret.GetData()

	if err != nil {
		logging.Default.Error(err, "Get secret error to read credential")
		return
	}

	if len(credentialSecretData) < 2 ||
		credentialSecretData["AccessKeyId"] == "" ||
		credentialSecretData["AccessKeySecret"] == "" {
		err = errors.New("secret is invalid. AccessKeyId and AccessKeySecret must be supplied")
		logging.Default.Error(err, "Read credential from secret error", "SecretName", secret.GetName())
		return
	}

	credential = &AliyunCredential{
		AccessKeyId:     credentialSecretData["AccessKeyId"],
		AccessKeySecret: credentialSecretData["AccessKeySecret"],
		SecurityToken:   credentialSecretData["SecurityToken"],
		Expiration:      credentialSecretData["Expiration"],
	}
	return
}
