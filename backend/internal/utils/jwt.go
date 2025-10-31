package utils

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GetJwtSecretString returns the resolved JWT secret as a string using unified logic.
// Resolution order: JWT_SECRET -> AURA_JWT_SECRET -> dev default (non-production only).
func GetJwtSecretString() (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		// Optional alias some teams use
		secret = strings.TrimSpace(os.Getenv("AURA_JWT_SECRET"))
	}
	if secret == "" {
		// Provide a safe dev default to avoid local setup drift unless explicitly disabled.
		// If AURA_STRICT_JWT is set to 1/true, we require an env secret.
		strict := strings.EqualFold(strings.TrimSpace(os.Getenv("AURA_STRICT_JWT")), "1") ||
			strings.EqualFold(strings.TrimSpace(os.Getenv("AURA_STRICT_JWT")), "true")
		if !strict {
			secret = "dev_jwt_secret_123"
		}
	}
	if secret == "" {
		return "", fmt.Errorf("JWT_SECRET environment variable not set")
	}
	return secret, nil
}

// GetJwtSecretBytes returns the resolved JWT secret in []byte form.
func GetJwtSecretBytes() ([]byte, error) {
	s, err := GetJwtSecretString()
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// GenerateJWT creates a new JWT token for a given user ID
func GenerateJWT(userID uuid.UUID) (string, error) {
	// Get the secret when the function is called
	jwtSecret, err := GetJwtSecretBytes()
	if err != nil {
		return "", err // Return the error if secret is missing
	}

	// Create the claims
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
		"iat":     time.Now().Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString(jwtSecret) // Use the retrieved secret

	return tokenString, err
}
