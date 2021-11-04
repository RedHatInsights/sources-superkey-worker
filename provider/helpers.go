package provider

import (
	"crypto/rand"
	"encoding/hex"
)

// generateGUID() generates a short guid for resources
func generateGUID() (string, error) {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
