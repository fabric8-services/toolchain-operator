package client

import (
	"context"
	configv1 "github.com/openshift/api/config/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
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
	Route
	Infrastructure
}

// Secret contains methods for manipulating Secrets
type Secret interface {
	GetSecret(namespace, name string) (*v1.Secret, error)
	CreateSecret(*v1.Secret) error
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
	CreateOAuthClient(*oauthv1.OAuthClient) error
	GetOAuthClient(name string) (*oauthv1.OAuthClient, error)
}

type Route interface {
	CreateRoute(route *routev1.Route) error
	GetRoute(namespace, name string) (*routev1.Route, error)
	DeleteRoute(r *routev1.Route) error
}

// Infrastructure contains method for manipulating Infrastructure
type Infrastructure interface {
	GetInfrastructure(name string) (*configv1.Infrastructure, error)
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

// CreateRoute creates the Route.
func (c *clientImpl) CreateRoute(r *routev1.Route) error {
	return c.Client.Create(context.Background(), r)
}

// GetRoute returns the existing Route.
func (c *clientImpl) GetRoute(namespace, name string) (*routev1.Route, error) {
	r := &routev1.Route{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, r); err != nil {
		return nil, err
	}
	return r, nil
}

// DeleteRoute deletes the Route.
func (c *clientImpl) DeleteRoute(r *routev1.Route) error {
	return c.Client.Delete(context.Background(), r)
}

// GetSecret returns the existing Secret.
func (c *clientImpl) GetSecret(namespace, name string) (*v1.Secret, error) {
	s := &v1.Secret{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, s); err != nil {
		return nil, err
	}
	return s, nil
}

// CreateSecret creates the Secret.
func (c *clientImpl) CreateSecret(s *v1.Secret) error {
	return c.Client.Create(context.Background(), s)
}

// GetInfrastructure returns the existing Infrastructure.
func (c *clientImpl) GetInfrastructure(name string) (*configv1.Infrastructure, error) {
	r := &configv1.Infrastructure{}
	if err := c.Client.Get(context.Background(), types.NamespacedName{Name: name}, r); err != nil {
		return nil, err
	}
	return r, nil
}
