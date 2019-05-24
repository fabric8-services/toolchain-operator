package config

import (
	"strings"

	"github.com/spf13/viper"
)

const (
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

	return c, nil
}
