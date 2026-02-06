package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// JWTPayload represents the decoded JWT payload
type JWTPayload struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Exp               int64  `json:"exp"`
	Iat               int64  `json:"iat"`
}

// DecodeJWT decodes a JWT token and returns its payload
func DecodeJWT(token string) (*JWTPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT format")
	}

	// Decode the payload (middle part)
	payload := parts[1]

	// Add padding if necessary
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try standard base64 as fallback
		decoded, err = base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, err
		}
	}

	var result JWTPayload
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// IsTokenExpired checks if a token is expired or will expire within the buffer period
func IsTokenExpired(token string) bool {
	if token == "" {
		return true
	}

	payload, err := DecodeJWT(token)
	if err != nil {
		return true
	}

	if payload.Exp == 0 {
		return true
	}

	// Check if token expires within 60 seconds (buffer)
	expirationTime := time.Unix(payload.Exp, 0)
	now := time.Now()
	buffer := 60 * time.Second

	return expirationTime.Before(now.Add(buffer))
}

// GetTokenExpiry returns the expiration time of a token
func GetTokenExpiry(token string) (time.Time, error) {
	payload, err := DecodeJWT(token)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(payload.Exp, 0), nil
}
