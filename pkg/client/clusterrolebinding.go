package client

import (
	"context"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
)

// CreateClusterRoleBinding creates the ClusterRoleBinding.
func (c *clientImpl) CreateClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) error {
	return c.Client.Create(context.Background(), crb)
}

// GetClusterRoleBinding returns the existing ClusteRoleBinding.
func (c *clientImpl) GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}

	if err := c.Client.Get(context.Background(), types.NamespacedName{Name: name}, crb); err != nil {
		return nil, err
	}
	return crb, nil
}
