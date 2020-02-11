package main

import (
	"flag"
	"os"

	rosapi "github.com/oam-dev/cloud-provider/alibabacloud/ros/apis/ros.alibabacloud.com/v1alpha1"
	rosclient "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/handlers"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/k8s"
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/logging"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
	"github.com/oam-dev/oam-go-sdk/pkg/oam"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = rosapi.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	// setup flag
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	var env string
	flag.StringVar(&env, "env", "test", "App running environment.")
	var endpoint string
	flag.StringVar(&endpoint, "endpoint", "https://ros.aliyuncs.com", "ROS api endpoint.")
	var regionId string
	flag.StringVar(&regionId, "region-id", "cn-hangzhou", "Region where ROS creates resources from.")
	var accessKeyId string
	flag.StringVar(&accessKeyId, "access-key-id", "", "User's access key ID.")
	var accessKeySecret string
	flag.StringVar(&accessKeySecret, "access-key-secret", "", "User's Access key secret.")
	var credentialSecretName string
	flag.StringVar(&credentialSecretName, "credential-secret-name", "", "User's credential secret name.")
	var leaderElectionNamespace string
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "default", "Leader election namespace.")
	var namespace string
	flag.StringVar(&namespace, "namespace", "default", "App namespace.")
	var updateApp bool
	flag.BoolVar(&updateApp, "update-app", false, "Whether update application status")
	var workAsRosCrd bool
	flag.BoolVar(&workAsRosCrd, "ros-crd", false, "whether this controller work as ROS or OAM CRD")
	flag.Parse()

	// init controller conf
	config.InitRosCtrlConf(env, endpoint, regionId, accessKeyId, accessKeySecret, credentialSecretName, leaderElectionNamespace, namespace, updateApp)

	// init log
	logging.Init()
	logging.SetUp.Info("ROS OAM controller stating")
	logging.SetUp.Info("Init ros-oam controller conf", "RosCtrlConf", config.RosCtrlConf)

	// init k8s client
	if err := k8s.Init(); err != nil {
		logging.SetUp.Error(err, "Problem occurs during stating ros controller")
		os.Exit(1)
	}
	logging.SetUp.Info("K8S client success initialized")

	// init manager
	options := ctrl.Options{
		Scheme:                  scheme,
		MetricsBindAddress:      metricsAddr,
		LeaderElection:          config.RosCtrlConf.LeaderElection,
		LeaderElectionID:        config.RosCtrlConf.LeaderLockName,
		LeaderElectionNamespace: config.RosCtrlConf.LeaderElectionNamespace,
		Namespace:               config.RosCtrlConf.Namespace,
	}
	oam.InitMgr(ctrl.GetConfigOrDie(), options)
	logging.SetUp.Info("Controller manager success initialized")

	// register hooks and handlers
	var option oam.Option
	if workAsRosCrd {
		oam.RegisterObject("rosstack", new(rosapi.RosStack))

		client, err := rosclient.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			logging.SetUp.Error(err, "Create ros runtime client err")
			os.Exit(1)
		}
		oam.RegisterHandlers("rosstack", &handlers.AppConfHandler{Name: "app", RosCrdClient: client})
		option = oam.WithSpec("rosstack")
	} else {
		client, err := versioned.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			logging.SetUp.Error(err, "Create oam runtime client err")
			os.Exit(1)
		}
		oam.RegisterHandlers(oam.STypeApplicationConfiguration, &handlers.AppConfHandler{Name: "app", OamCrdClient: client})
		option = oam.WithApplicationConfiguration()
	}

	logging.SetUp.Info("Add hooks and handlers success")

	if err := oam.Run(option); err != nil {
		logging.SetUp.Error(err, "Problem occurs during stating ros controller")
		os.Exit(1)
	}
}
