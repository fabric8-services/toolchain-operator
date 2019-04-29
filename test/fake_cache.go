package test

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FakeCache struct {
	Err error
}

func (c *FakeCache) GetInformer(obj runtime.Object) (toolscache.SharedIndexInformer, error) {
	return nil, nil
}

func (c *FakeCache) GetInformerForKind(gvk schema.GroupVersionKind) (toolscache.SharedIndexInformer, error) {
	return nil, nil

}

func (c *FakeCache) Start(stopCh <-chan struct{}) error {
	return nil
}

func (c *FakeCache) WaitForCacheSync(stop <-chan struct{}) bool {
	return true
}

func (c *FakeCache) IndexField(obj runtime.Object, field string, extractValue client.IndexerFunc) error {
	return nil
}

func (c *FakeCache) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return c.Err
}

func (c *FakeCache) List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
	return nil
}

func NewFakeCache(err error) cache.Cache {
	return &FakeCache{Err: err}
}
