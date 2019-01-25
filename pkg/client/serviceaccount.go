package client

import (
	"context"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CreateServiceAccount creates the serviceAccount.
func (c *clientImpl) CreateServiceAccount(sa *v1.ServiceAccount) error {
	return c.Client.Create(context.Background(), sa)
}

// GetServiceAccount returns the existing serviceAccount.
func (c *clientImpl) GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error) {
	sa := &v1.ServiceAccount{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, sa); err != nil {
		return nil, err
	}
	return sa, nil
}
