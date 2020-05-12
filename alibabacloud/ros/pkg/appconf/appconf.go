//go:generate mockgen -destination mock_appconf.go -package appconf -source appconf.go
package appconf

import (
	"errors"
	rosv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/apis/ros.alibabacloud.com/v1alpha1"
	roscrd "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	oam "github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	oamv1alpha1 "github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppConfInterface interface {
	ToOamApplicationConfiguration() *oamv1alpha1.ApplicationConfiguration
	ToRosStack() *rosv1alpha1.RosStack
	ToObject() v1.Object
	Update(c *Context, appConf AppConfInterface) (err error)
	UpdateStatus(
		c *Context,
		phase v1alpha1.ApplicationPhase,
		type_ v1alpha1.ApplicationConditionType,
		message string) (err error)
	GetScopes() []v1alpha1.ScopeBinding
	GetName() string
	GetNamespace() string
	GetObjectMeta() v1.ObjectMeta
}

type AppConf struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	//in our case, we use the same spec with applicationConfiguration
	Spec   oam.ApplicationConfigurationSpec   `json:"spec,omitempty"`
	Status oam.ApplicationConfigurationStatus `json:"status,omitempty"`

	isOamAppConf bool
	oamAppConf   *oamv1alpha1.ApplicationConfiguration
	rosStack     *rosv1alpha1.RosStack
}

type AppConfFromContextGetterFunc func(c *Context) (appConf AppConfInterface, err error)

type AppConfGetterFunc func(
	AppConfNamespace string,
	AppConfName string,
	OamCrdClient *versioned.Clientset,
	RosCrdClient *roscrd.Clientset,
) (appConf AppConfInterface, err error)

func GetAppConf(
	AppConfNamespace string,
	AppConfName string,
	OamCrdClient *versioned.Clientset,
	RosCrdClient *roscrd.Clientset,
) (appConf AppConfInterface, err error) {
	if OamCrdClient != nil {
		appConfInterface := OamCrdClient.CoreV1alpha1().ApplicationConfigurations(AppConfNamespace)
		appConf_, err := appConfInterface.Get(AppConfName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		appConf, _ = NewAppConf(appConf_)
		return appConf, nil
	} else if RosCrdClient != nil {
		appConfInterface := RosCrdClient.RosV1alpha1().RosStacks(AppConfNamespace)
		appConf_, err := appConfInterface.Get(AppConfName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		appConf, _ = NewAppConf(appConf_)
		return appConf, nil
	} else {
		return nil, errors.New("no client found")
	}
}

func GetAppConfFromContext(c *Context) (appConf AppConfInterface, err error) {
	return GetAppConf(
		c.AppConf.GetNamespace(),
		c.AppConf.GetName(),
		c.OamCrdClient,
		c.RosCrdClient)
}

func NewAppConf(ac interface{}) (appConf *AppConf, ok bool) {
	switch typedAppConf := ac.(type) {
	case *oamv1alpha1.ApplicationConfiguration:
		appConf = &AppConf{
			TypeMeta:     typedAppConf.TypeMeta,
			ObjectMeta:   typedAppConf.ObjectMeta,
			Spec:         typedAppConf.Spec,
			Status:       typedAppConf.Status,
			isOamAppConf: true,
			oamAppConf:   typedAppConf,
		}
		return appConf, true
	case *rosv1alpha1.RosStack:
		appConf = &AppConf{
			TypeMeta:     typedAppConf.TypeMeta,
			ObjectMeta:   typedAppConf.ObjectMeta,
			Spec:         typedAppConf.Spec,
			Status:       typedAppConf.Status,
			isOamAppConf: false,
			rosStack:     typedAppConf,
		}
		return appConf, true
	default:
		return nil, false
	}
}

func (a *AppConf) ToOamApplicationConfiguration() *oamv1alpha1.ApplicationConfiguration {
	a.oamAppConf.TypeMeta = a.TypeMeta
	a.oamAppConf.ObjectMeta = a.ObjectMeta
	a.oamAppConf.Spec = a.Spec
	a.oamAppConf.Status = a.Status
	return a.oamAppConf
}

func (a *AppConf) ToRosStack() *rosv1alpha1.RosStack {
	a.rosStack.TypeMeta = a.TypeMeta
	a.rosStack.ObjectMeta = a.ObjectMeta
	a.rosStack.Spec = a.Spec
	a.rosStack.Status = a.Status
	return a.rosStack
}

func (a *AppConf) ToObject() v1.Object {
	return interface{}(a).(v1.Object)
}

func (a *AppConf) Update(c *Context, appConf AppConfInterface) (err error) {
	err = a.checkContext(c)
	if err != nil {
		return
	}

	if c.OamCrdClient != nil {
		appConfInterface := c.OamCrdClient.CoreV1alpha1().ApplicationConfigurations(appConf.GetNamespace())
		_, err = appConfInterface.Update(appConf.ToOamApplicationConfiguration())
		if err == nil {
			c.AppConf = appConf
		}
		return
	} else if c.RosCrdClient != nil {
		appConfInterface := c.RosCrdClient.RosV1alpha1().RosStacks(appConf.GetNamespace())
		_, err = appConfInterface.Update(appConf.ToRosStack())
		if err == nil {
			c.AppConf = appConf
		}
		return
	}
	return
}

func (a *AppConf) UpdateStatus(
	c *Context,
	phase v1alpha1.ApplicationPhase, type_ v1alpha1.ApplicationConditionType,
	message string) (err error) {

	err = a.checkContext(c)
	if err != nil {
		return
	}

	if a.Status.Conditions == nil {
		condition := v1alpha1.ApplicationCondition{
			Type:               type_,
			Status:             corev1.ConditionTrue,
			LastUpdateTime:     v1.Now(),
			LastTransitionTime: v1.Now(),
			Reason:             message,
			Message:            message,
		}

		a.Status = v1alpha1.ApplicationConfigurationStatus{
			Phase:      phase,
			Conditions: []v1alpha1.ApplicationCondition{condition},
		}
	} else {
		a.Status.Phase = phase
		condition := &a.Status.Conditions[0]
		condition.Type = type_
		condition.LastUpdateTime = v1.Now()
		condition.LastTransitionTime = v1.Now()
		if message != "" {
			condition.Message = message
		}
	}

	if c.OamCrdClient != nil {
		appConfInterface := c.OamCrdClient.CoreV1alpha1().ApplicationConfigurations(a.Namespace)
		_, err = appConfInterface.UpdateStatus(a.ToOamApplicationConfiguration())
	} else {
		appConfInterface := c.RosCrdClient.RosV1alpha1().RosStacks(a.Namespace)
		_, err = appConfInterface.UpdateStatus(a.ToRosStack())
	}

	if err == nil {
		c.AppConf = a
	}

	return
}

func (a *AppConf) GetScopes() []v1alpha1.ScopeBinding {
	return a.Spec.Scopes
}

func (a *AppConf) GetName() string {
	return a.Name
}

func (a *AppConf) GetNamespace() string {
	return a.Namespace
}

func (a *AppConf) GetObjectMeta() v1.ObjectMeta {
	return a.ObjectMeta
}

func (a *AppConf) checkContext(c *Context) (err error) {
	if c.OamCrdClient == nil && c.RosCrdClient == nil {
		return errors.New("no client found in context")
	}
	return nil
}
