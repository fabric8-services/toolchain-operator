package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	AuthURL         = "auth.url"
	ClusterURL      = "cluster.url"
	TCClientID      = "tc.client.id"
	TCClientSecret  = "tc.client.secret"
	ClusterName     = "cluster.name"
	Name            = "toolchain-enabler"
	SAName          = "toolchain-sre"
	OAuthClientName = "codeready-toolchain"
)

// Configuration encapsulates the Viper configuration object which stores the configuration data.
type Configuration struct {
	v *viper.Viper
}

// New creates a configuration reader object using Env Variables
func NewConfiguration() (*Configuration, error) {
	c := &Configuration{
		v: viper.New(),
	}

	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := c.validateURL(c.GetAuthServiceURL(), "auth service"); err != nil {
		return c, err
	}

	if err := c.validateURL(c.GetClusterServiceURL(), "cluster service"); err != nil {
		return c, err
	}

	return c, nil
}

// returns the hostname of the given URL if this latter was not empty
func (c *Configuration) validateURL(serviceURL, serviceName string) error {
	if serviceURL == "" {
		return errors.New(fmt.Sprintf("%s url is empty", serviceName))
	} else {
		u, err := url.Parse(serviceURL)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf("invalid url for %s: %s", serviceName, serviceURL))
		}
		if u.Host == "" {
			return errors.New(fmt.Sprintf("invalid url '%s' (missing scheme or host?) for: %s", serviceURL, serviceName))
		}
	}
	return nil
}

// GetAuthServiceURL returns Auth Service URL
func (c *Configuration) GetAuthServiceURL() string {
	return c.v.GetString(AuthURL)
}

// GetClusterServiceURL returns Cluster Service URL
func (c *Configuration) GetClusterServiceURL() string {
	return c.v.GetString(ClusterURL)
}

// GetClientID return toolchain client-id required to create SA token
func (c *Configuration) GetClientID() string {
	return c.v.GetString(TCClientID)
}

// GetClientSecret return toolchain secret required to create SA token
func (c *Configuration) GetClientSecret() string {
	return c.v.GetString(TCClientSecret)
}

// // GetClusterName returns cluster name confifured while creating cluster
func (c *Configuration) GetClusterName() string {
	return c.v.GetString(ClusterName)
}
