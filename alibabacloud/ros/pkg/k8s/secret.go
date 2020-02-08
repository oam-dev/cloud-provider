package k8s

import (
	"github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func GetSecretData(secretName string) (data map[string]string, err error) {
	secretName = strings.ToLower(secretName)
	data = make(map[string]string)
	secret, err := ClientManager.Clientset.
		CoreV1().
		Secrets(config.RosCtrlConf.Namespace).
		Get(secretName, metav1.GetOptions{})
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

func UpdateSecretData(secretName string, secretData map[string]string) (err error) {
	secretName = strings.ToLower(secretName)
	secretInterface := ClientManager.Clientset.CoreV1().Secrets(config.RosCtrlConf.Namespace)
	secret, err := secretInterface.Get(secretName, metav1.GetOptions{})

	byteData := make(map[string][]byte)
	for key, value := range secretData {
		byteData[key] = []byte(value)
	}

	if errors.IsNotFound(err) {
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: config.RosCtrlConf.Namespace,
			},

			Data: byteData,
			Type: "Opaque",
		}
		err = nil
		_, err = secretInterface.Create(secret)
	} else if err == nil {
		for key, value := range byteData {
			secret.Data[key] = value
		}
		_, err = secretInterface.Update(secret)
	}
	return
}

func SetSecretData(secretName string, secretData map[string]string) (err error) {
	secretName = strings.ToLower(secretName)
	secretInterface := ClientManager.Clientset.CoreV1().Secrets(config.RosCtrlConf.Namespace)
	secret, err := secretInterface.Get(secretName, metav1.GetOptions{})

	byteData := make(map[string][]byte)
	for key, value := range secretData {
		byteData[key] = []byte(value)
	}

	secret = &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "apps/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
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

func DeleteSecretData(secretName string) (err error) {
	secretName = strings.ToLower(secretName)
	err = ClientManager.Clientset.
		CoreV1().
		Secrets(config.RosCtrlConf.Namespace).
		Delete(secretName, &metav1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return
}
