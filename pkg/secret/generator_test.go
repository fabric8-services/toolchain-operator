package secret

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateRandomString(t *testing.T) {
	bits, err := CreateRandomString(256)

	require.NoError(t, err)
	require.NotEmpty(t, bits)
}
