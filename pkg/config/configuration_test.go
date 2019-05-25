package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateURL(t *testing.T) {

	t.Run("valid url", func(t *testing.T) {

		t.Run("auth and cluster", func(t *testing.T) {
			err := validateURL("http://auth", "auth service")
			assert.NoError(t, err)

			err = validateURL("http://cluster", "cluster service")
			assert.NoError(t, err)
		})

	})

	t.Run("invalid url", func(t *testing.T) {

		t.Run("missing scheme", func(t *testing.T) {
			err := validateURL("auth", "auth service")
			require.EqualError(t, err, "invalid url 'auth' (missing scheme or host?) for: auth service")
		})

		t.Run("missing host", func(t *testing.T) {
			err := validateURL("http://", "auth service")
			require.EqualError(t, err, "invalid url 'http://' (missing scheme or host?) for: auth service")
		})

		t.Run("auth empty", func(t *testing.T) {
			err := validateURL("", "auth service")
			require.EqualError(t, err, "'auth service' url is empty")
		})

	})

}
