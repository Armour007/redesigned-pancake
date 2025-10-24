package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// getJwtSecret retrieves the JWT secret from environment variables
func getJwtSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Return an error if the secret is not found
		return nil, fmt.Errorf("JWT_SECRET environment variable not set")
	}
	return []byte(secret), nil
}

// GenerateJWT creates a new JWT token for a given user ID
func GenerateJWT(userID uuid.UUID) (string, error) {
	// Get the secret when the function is called
	jwtSecret, err := getJwtSecret()
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
