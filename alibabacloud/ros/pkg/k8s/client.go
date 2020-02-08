package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	DefaultBurstForMaster = rest.DefaultBurst * 10
	DefaultQPSForMaster   = rest.DefaultQPS * 10
)

// Client Manager
var ClientManager = clientManager{}

type clientManager struct {
	Clientset kubernetes.Interface
}

func Init() error {
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return err
	}
	cfg.Burst = DefaultBurstForMaster
	cfg.QPS = DefaultQPSForMaster

	ClientManager.Clientset = kubernetes.NewForConfigOrDie(cfg)
	return nil
}
