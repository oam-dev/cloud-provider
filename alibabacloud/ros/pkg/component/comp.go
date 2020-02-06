package component

import (
	"context"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/oam"
	"k8s.io/apimachinery/pkg/types"
)

// get component schematic by namespace and name
func Get(namespace, name string) (*v1alpha1.ComponentSchematic, error) {
	comp := &v1alpha1.ComponentSchematic{}
	if err := oam.GetMgr().GetClient().Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, comp); err != nil {
		return nil, err
	}
	return comp, nil
}
