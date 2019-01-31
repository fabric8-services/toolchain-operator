package client

import (
	"context"
	apioauthv1 "github.com/openshift/api/oauth/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	client.Client
	Secret
	ServiceAccount
	ClusterRoleBinding
	OAuthClient
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

// OAuthClient contains methods for manipulating OAuthClient.
type OAuthClient interface {
	CreateOAuthClient(*apioauthv1.OAuthClient) error
	GetOAuthClient(name string) (*apioauthv1.OAuthClient, error)
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

// CreateOauthClient creates the OauthClient.
func (c *clientImpl) CreateOAuthClient(oc *oauthv1.OAuthClient) error {
	return c.Client.Create(context.Background(), oc)
}

// GetOauthClient returns the existing OAuthClient.
func (c *clientImpl) GetOAuthClient(name string) (*oauthv1.OAuthClient, error) {
	oc := &oauthv1.OAuthClient{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Name: name}, oc); err != nil {
		return nil, err
	}
	return oc, nil
}

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

// GetSecret returns the existing Secret.
func (c *clientImpl) GetSecret(namespace, name string) (*v1.Secret, error) {
	s := &v1.Secret{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, s); err != nil {
		return nil, err
	}
	return s, nil
}
