package component

import (
	"context"
	"github.com/oam-dev/oam-go-sdk/apis/core.oam.dev/v1alpha1"
	"github.com/oam-dev/oam-go-sdk/pkg/oam"
	"k8s.io/apimachinery/pkg/types"
)

// Get takes namespace and name. Returns component schematic and an error if there is any.
func Get(namespace, name string) (*v1alpha1.ComponentSchematic, error) {
	comp := &v1alpha1.ComponentSchematic{}
	backgroud := context.Background()
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	if err := oam.GetMgr().GetClient().Get(backgroud, namespacedName, comp); err != nil {
		return nil, err
	}
	return comp, nil
}
