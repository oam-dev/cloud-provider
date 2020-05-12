//go:generate mockgen -destination mock_secret.go -package k8s -source secret.go
package k8s

import (
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// secretOption defines Secret option
type secretOption struct {
	ClientSet kubernetes.Interface
}

// SecretOption has methods to work with Secret option.
type SecretOption interface {
	apply(*secretOption)
}

// funcOption defines function used for Secret option
type funcOption struct {
	f func(*secretOption)
}

// apply executes funcOption's func
func (fdo *funcOption) apply(do *secretOption) {
	fdo.f(do)
}

// newFuncOption returns function option
func newFuncOption(f func(*secretOption)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// WithClientSet sets client set in Secret option
func WithClientSet(clientSet kubernetes.Interface) SecretOption {
	return newFuncOption(func(o *secretOption) {
		o.ClientSet = clientSet
	})
}

// SecretInterface has methods to work with Secret resources.
type SecretInterface interface {
	GetName() string
	GetData() (data map[string]string, err error)
	UpdateData(data map[string]string) (err error)
	SetData(data map[string]string) (err error)
	DeleteData() (err error)
}

// Secret implements SecretInterface
type Secret struct {
	Name      string
	clientSet kubernetes.Interface
}

// NewSecret returns a Secret
func NewSecret(name string, opts ...SecretOption) SecretInterface {
	// init options
	o := &secretOption{}
	for _, opt := range opts {
		opt.apply(o)
	}

	if o.ClientSet == nil {
		o.ClientSet = ClientManager.Clientset
	}

	// new Secret
	name = strings.ToLower(name)
	return &Secret{
		Name:      name,
		clientSet: o.ClientSet,
	}
}

// GetName returns the Secret name.
func (c *Secret) GetName() string {
	return c.Name
}

// GetData returns the corresponding Secret data, and an error if there is any.
func (c *Secret) GetData() (data map[string]string, err error) {
	data = make(map[string]string)
	secret, err := c.clientSet.
		CoreV1().
		Secrets(config.RosCtrlConf.Namespace).
		Get(c.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return data, nil
	}

	byteData := secret.Data
	if byteData != nil {
		for key, value := range byteData {
			data[key] = string(value)
		}
	}
	return
}

// UpdateData takes data and updates it. Returns an error if one occurs.
func (c *Secret) UpdateData(data map[string]string) (err error) {
	secretInterface := c.clientSet.CoreV1().Secrets(config.RosCtrlConf.Namespace)
	secret, err := secretInterface.Get(c.Name, metav1.GetOptions{})

	byteData := make(map[string][]byte)
	for key, value := range data {
		byteData[key] = []byte(value)
	}

	if errors.IsNotFound(err) {
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Name,
				Namespace: config.RosCtrlConf.Namespace,
			},

			Data: byteData,
			Type: "Opaque",
		}
		err = nil
		_, err = secretInterface.Create(secret)
	} else if err == nil {
		if secret.Data == nil {
			secret.Data = byteData
		} else {
			for key, value := range byteData {
				secret.Data[key] = value
			}
		}
		_, err = secretInterface.Update(secret)
	}
	return
}

// SetData takes data and sets it. Returns an error if one occurs.
func (c *Secret) SetData(data map[string]string) (err error) {
	secretInterface := c.clientSet.CoreV1().Secrets(config.RosCtrlConf.Namespace)
	secret, err := secretInterface.Get(c.Name, metav1.GetOptions{})

	byteData := make(map[string][]byte)
	for key, value := range data {
		byteData[key] = []byte(value)
	}

	secret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "apps/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: config.RosCtrlConf.Namespace,
		},

		Data: byteData,
		Type: "Opaque",
	}

	if errors.IsNotFound(err) {
		_, err = secretInterface.Create(secret)
	} else if err == nil {
		_, err = secretInterface.Update(secret)
	}
	return
}

// DeleteData deletes data. Returns an error if one occurs.
func (c *Secret) DeleteData() (err error) {
	err = c.clientSet.
		CoreV1().
		Secrets(config.RosCtrlConf.Namespace).
		Delete(c.Name, &metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return
}
