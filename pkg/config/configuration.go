package config

import (
	"fmt"
	"net/url"

	codereadyv1alpha1 "github.com/fabric8-services/toolchain-operator/pkg/apis/codeready/v1alpha1"
	errs "github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const (
	TCClientID      = "tc.client.id"
	TCClientSecret  = "tc.client.secret"
	Name            = "toolchain-enabler"
	SAName          = "toolchain-sre"
	OAuthClientName = "codeready-toolchain"
)

type ToolchainConfig struct {
	AuthURL      string
	ClusterURL   string
	ClusterName  string
	ClientID     string
	ClientSecret string
}

func (c ToolchainConfig) GetClusterServiceURL() string {
	return c.ClusterURL
}

func (c ToolchainConfig) GetAuthServiceURL() string {
	return c.AuthURL
}

func (c ToolchainConfig) GetClientID() string {
	return c.ClientID
}

func (c ToolchainConfig) GetClientSecret() string {
	return c.ClientSecret
}

func (c ToolchainConfig) GetClusterName() string {
	return c.ClusterName
}

func Create(spec codereadyv1alpha1.ToolChainEnablerSpec, secret *v1.Secret) (tcConfig ToolchainConfig, err error) {
	if err = validateURL(spec.AuthURL, "auth service"); err != nil {
		return tcConfig, err
	}
	if err = validateURL(spec.ClusterURL, "cluster service"); err != nil {
		return tcConfig, err
	}
	if len(secret.Data[TCClientID]) <= 0 {
		return tcConfig, errs.New(fmt.Sprintf("'%s' is empty in secret '%s'", TCClientID, spec.ToolchainSecretName))
	}
	if len(secret.Data[TCClientSecret]) <= 0 {
		return tcConfig, errs.New(fmt.Sprintf("'%s' is empty in secret '%s'", TCClientSecret, spec.ToolchainSecretName))
	}

	tcConfig = ToolchainConfig{
		AuthURL:      spec.AuthURL,
		ClusterURL:   spec.ClusterURL,
		ClusterName:  spec.ClusterName,
		ClientID:     string(secret.Data[TCClientID]),
		ClientSecret: string(secret.Data[TCClientSecret]),
	}
	return tcConfig, nil
}

func validateURL(serviceURL, serviceName string) error {
	if serviceURL == "" {
		return errs.New(fmt.Sprintf("'%s' url is empty", serviceName))
	} else {
		u, err := url.Parse(serviceURL)
		if err != nil {
			return errs.Wrapf(err, fmt.Sprintf("invalid url for %s: '%s'", serviceName, serviceURL))
		}
		if u.Host == "" {
			return errs.New(fmt.Sprintf("invalid url '%s' (missing scheme or host?) for: %s", serviceURL, serviceName))
		}
	}
	return nil
}
