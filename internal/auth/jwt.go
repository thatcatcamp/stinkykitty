// SPDX-License-Identifier: MIT
package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// Claims represents JWT claims for authentication
type Claims struct {
	UserID        uint   `json:"user_id"`
	Email         string `json:"email"`
	SiteID        uint   `json:"site_id"`
	IsGlobalAdmin bool   `json:"is_global_admin"`
	jwt.RegisteredClaims
}

// getJWTSecret returns the JWT secret from env var or config
func getJWTSecret() string {
	// Environment variable takes precedence
	if secret := os.Getenv("STINKY_JWT_SECRET"); secret != "" {
		return secret
	}
	return config.GetString("auth.jwt_secret")
}

// GenerateToken creates a JWT token for a user and site
func GenerateToken(user *models.User, site *models.Site) (string, error) {
	expiryHours := config.GetInt("auth.jwt_expiry_hours")
	if expiryHours == 0 {
		expiryHours = 8 // Default fallback
	}

	claims := Claims{
		UserID:        user.ID,
		Email:         user.Email,
		SiteID:        site.ID,
		IsGlobalAdmin: user.IsGlobalAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// ValidateToken parses and validates a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(getJWTSecret()), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
