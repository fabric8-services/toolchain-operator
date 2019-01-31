package secret

import (
	"crypto/rand"
	"encoding/base64"
)

// CreateRandomString returns a random string with at least the requested bits.
func CreateRandomString(bits int) (string, error) {
	size := bits / 8
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
