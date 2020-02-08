/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	rosalibabacloudcomv1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/apis/ros.alibabacloud.com/v1alpha1"
	versioned "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/clientset/versioned"
	internalinterfaces "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/oam-dev/cloud-provider/alibabacloud/ros/pkg/client/listers/ros.alibabacloud.com/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// RosStackInformer provides access to a shared informer and lister for
// RosStacks.
type RosStackInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.RosStackLister
}

type rosStackInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewRosStackInformer constructs a new informer for RosStack type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRosStackInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRosStackInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredRosStackInformer constructs a new informer for RosStack type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRosStackInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.RosV1alpha1().RosStacks(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.RosV1alpha1().RosStacks(namespace).Watch(options)
			},
		},
		&rosalibabacloudcomv1alpha1.RosStack{},
		resyncPeriod,
		indexers,
	)
}

func (f *rosStackInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRosStackInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *rosStackInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&rosalibabacloudcomv1alpha1.RosStack{}, f.defaultInformer)
}

func (f *rosStackInformer) Lister() v1alpha1.RosStackLister {
	return v1alpha1.NewRosStackLister(f.Informer().GetIndexer())
}