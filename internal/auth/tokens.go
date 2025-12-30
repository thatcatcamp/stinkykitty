// SPDX-License-Identifier: MIT
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateResetToken creates a password reset token
func GenerateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
