package config

import (
	. "github.com/fabric8-services/toolchain-operator/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfiguration(t *testing.T) {

	t.Run("valid url", func(t *testing.T) {

		t.Run("auth and cluster", func(t *testing.T) {
			reset := SetEnv(Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://cluster"))
			defer reset()

			c, err := NewConfiguration()
			require.NoError(t, err)

			assert.Equal(t, c.GetAuthServiceURL(), "http://auth")
			assert.Equal(t, c.GetClusterServiceURL(), "http://cluster")
		})

	})

	t.Run("invalid url", func(t *testing.T) {

		t.Run("missing scheme", func(t *testing.T) {
			reset := SetEnv(Env("AUTH_URL", "auth"), Env("CLUSTER_URL", "http://cluster"))
			defer reset()

			_, err := NewConfiguration()
			require.EqualError(t, err, "invalid url 'auth' (missing scheme or host?) for: auth service")
		})

		t.Run("missing host", func(t *testing.T) {
			reset := SetEnv(Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://"))
			defer reset()

			_, err := NewConfiguration()
			require.EqualError(t, err, "invalid url 'http://' (missing scheme or host?) for: cluster service")
		})

		t.Run("auth empty", func(t *testing.T) {
			reset := SetEnv(Env("AUTH_URL", ""), Env("CLUSTER_URL", "http://"))
			defer reset()

			_, err := NewConfiguration()
			require.EqualError(t, err, "auth service url is empty")
		})

		t.Run("cluster empty", func(t *testing.T) {
			reset := SetEnv(Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", ""))
			defer reset()

			_, err := NewConfiguration()
			require.EqualError(t, err, "cluster service url is empty")
		})
	})

	t.Run("client id and secret", func(t *testing.T) {
		reset := SetEnv(Env("TC_CLIENT_ID", "toolchain"), Env("TC_CLIENT_SECRET", "secret"), Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://cluster"))
		defer reset()

		c, err := NewConfiguration()
		require.NoError(t, err)

		assert.Equal(t, c.GetClientID(), "toolchain")
		assert.Equal(t, c.GetClientSecret(), "secret")
	})

	t.Run("cluster name", func(t *testing.T) {
		reset := SetEnv(Env("CLUSTER_NAME", "dsaas-stage"), Env("AUTH_URL", "http://auth"), Env("CLUSTER_URL", "http://cluster"))
		defer reset()

		c, err := NewConfiguration()
		require.NoError(t, err)

		assert.Equal(t, c.GetClusterName(), "dsaas-stage")
	})
}
