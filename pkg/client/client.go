package client

import (
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	client.Client
	Secret
	ServiceAccount
	ClusterRoleBinding
}

// Secret contains methods for manipulating Secrets
type Secret interface {
	GetSecret(namespace, name string) (*v1.Secret, error)
}

// ServiceAccount contains methods for manipulating ServiceAccounts.
type ServiceAccount interface {
	CreateServiceAccount(*v1.ServiceAccount) error
	GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error)
}

// ClusterRoleBinding contains methods for manipulating ClusterRoleBindings.
type ClusterRoleBinding interface {
	CreateClusterRoleBinding(*rbacv1.ClusterRoleBinding) error
	GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error)
}

// Interface assertion.
var _ Client = &clientImpl{}

// clientImpl is a kubernetes client that can talk to the API server.
type clientImpl struct {
	client.Client
}

// NewClient creates a kubernetes client
func NewClient(k8sClient client.Client) Client {
	return &clientImpl{k8sClient}
}
