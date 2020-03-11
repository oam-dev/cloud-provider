package config

import (
	"os"
	"strings"
)

const (
	TEST_ENV               = "test"
	PRODUCTION_ENV         = "production"
	RESOURCE_IDENTITY      = "resource-identity"
	RESOURCE_IDENTITY_TYPE = "oam.alibaba.dev/v1.ResourceIdentity"
	ROS_GROUP              = "ros.aliyun.com"
	ROS_FINALIZER          = "ros.aliyun.com/ros-finalizer"
	BASE_USER_AGENT        = "ros-oam"
)

var (
	RosCtrlConf = RosControllerConfig{}
)

type RosControllerConfig struct {
	// HA
	LeaderElection          bool
	LeaderLockName          string
	LeaderElectionNamespace string

	Namespace string

	// Env
	Env       string
	UserAgent string

	// Log
	LoggerDebug bool
	LogToFile   bool
	LogFilePath string

	// API
	Endpoint string
	RegionId string

	// AK
	AccessKeyId          string
	AccessKeySecret      string
	CredentialSecretName string

	// Lifecycle
	UpdateApp          bool
	StackCheckInterval int

	// dryRun
	DryRun bool
}

func InitRosCtrlConf(
	env string,
	endpoint string,
	regionId string,
	accessKeyId string,
	accessKeySecret string,
	credentialSecretName string,
	leaderElectionNamespace string,
	namespace string,
	updateApp bool,
	serviceUserAgent string,
	dryRun bool) {

	RosCtrlConf.Env = env
	RosCtrlConf.StackCheckInterval = 5
	RosCtrlConf.UpdateApp = updateApp

	if endpoint != "" {
		RosCtrlConf.Endpoint = endpoint
	}

	if regionId != "" {
		RosCtrlConf.RegionId = regionId
	}

	RosCtrlConf.DryRun = dryRun

	// dry run don't do real action, so we don't need ak/sk here.
	if !dryRun {
		RosCtrlConf.AccessKeyId = accessKeyId
		if accessKeyId == "" {
			RosCtrlConf.AccessKeyId = os.Getenv("ACCESS_KEY_ID")
		}

		RosCtrlConf.AccessKeySecret = accessKeySecret
		if accessKeySecret == "" {
			RosCtrlConf.AccessKeySecret = os.Getenv("ACCESS_KEY_SECRET")
		}

		RosCtrlConf.CredentialSecretName = credentialSecretName
		if credentialSecretName == "" {
			RosCtrlConf.CredentialSecretName = os.Getenv("CREDENTIAL_SECRET_NAME")
		}
	}

	RosCtrlConf.LeaderElectionNamespace = leaderElectionNamespace
	if leaderElectionNamespace == "" {
		RosCtrlConf.LeaderElectionNamespace = os.Getenv("LEADER_ELECTION_NAMESPACE")
	}

	RosCtrlConf.Namespace = namespace
	if namespace == "" {
		RosCtrlConf.Namespace = os.Getenv("NAMESPACE")
	}

	RosCtrlConf.UserAgent = BASE_USER_AGENT
	if serviceUserAgent != "" {
		RosCtrlConf.UserAgent = BASE_USER_AGENT + ":" + strings.ReplaceAll(serviceUserAgent, " ", "-")
	}

	// controller options, log settings
	if env == PRODUCTION_ENV {
		RosCtrlConf.LeaderElection = true
		RosCtrlConf.LeaderLockName = "ros-oam-controller-lock"

		RosCtrlConf.LoggerDebug = false
		RosCtrlConf.LogToFile = true
		RosCtrlConf.LogFilePath = "/var/log/ros/controller.log"

	} else {
		RosCtrlConf.LeaderElection = false

		RosCtrlConf.LoggerDebug = true
		RosCtrlConf.LogToFile = false
	}

}
