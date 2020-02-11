package v1alpha1

import (
	rosv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/apis/ros.alibabacloud.com/v1alpha1"
	oam "github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	oamv1alpha1 "github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ApplicationConfiguration struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	//in our case, we use the same spec with applicationConfiguration
	Spec   oam.ApplicationConfigurationSpec   `json:"spec,omitempty"`
	Status oam.ApplicationConfigurationStatus `json:"status,omitempty"`

	isOamAppConf bool
	oamAppConf   *oamv1alpha1.ApplicationConfiguration
	rosStack     *rosv1alpha1.RosStack
}

func NewApplicationConfiguration(ac interface{}) (appConf *ApplicationConfiguration, ok bool) {
	switch typedAppConf := ac.(type) {
	case *oamv1alpha1.ApplicationConfiguration:
		appConf = &ApplicationConfiguration{
			TypeMeta:     typedAppConf.TypeMeta,
			ObjectMeta:   typedAppConf.ObjectMeta,
			Spec:         typedAppConf.Spec,
			Status:       typedAppConf.Status,
			isOamAppConf: true,
			oamAppConf:   typedAppConf,
		}
		return appConf, true
	case *rosv1alpha1.RosStack:
		appConf = &ApplicationConfiguration{
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

func (a *ApplicationConfiguration) ToOamApplicationConfiguration() *oamv1alpha1.ApplicationConfiguration {
	a.oamAppConf.TypeMeta = a.TypeMeta
	a.oamAppConf.ObjectMeta = a.ObjectMeta
	a.oamAppConf.Spec = a.Spec
	a.oamAppConf.Status = a.Status
	return a.oamAppConf
}

func (a *ApplicationConfiguration) ToRosStack() *rosv1alpha1.RosStack {
	a.rosStack.TypeMeta = a.TypeMeta
	a.rosStack.ObjectMeta = a.ObjectMeta
	a.rosStack.Spec = a.Spec
	a.rosStack.Status = a.Status
	return a.rosStack
}

func (a *ApplicationConfiguration) ToObject() v1.Object {
	return interface{}(a).(v1.Object)
}
