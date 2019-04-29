package test

import (
	"errors"
	"github.com/fabric8-services/toolchain-operator/pkg/client"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type DummyClient struct {
	client.Client
	resources map[string]string
}

func NewDummyClient(k8sClient client.Client, opts map[string]string) client.Client {
	return &DummyClient{k8sClient, opts}
}

func (d *DummyClient) GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error) {
	if msg, ok := d.resources["sa"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetServiceAccount(namespace, name)
}

func (d *DummyClient) GetClusterRoleBinding(name string) (*rbacv1.ClusterRoleBinding, error) {
	if msg, ok := d.resources["crb"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetClusterRoleBinding(name)
}

func (d *DummyClient) GetOAuthClient(name string) (*oauthv1.OAuthClient, error) {
	if msg, ok := d.resources["oc"]; ok {
		return nil, errors.New(msg)
	}
	return d.Client.GetOAuthClient(name)
}
