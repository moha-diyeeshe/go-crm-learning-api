package auth // Provides JWT creation and validation helpers for authenticated routes.

import (
	"fmt"  // Uses fmt.Errorf to wrap JWT parse/sign errors.
	"time" // Uses time.Now and time.Duration for token expiry handling.

	"github.com/golang-jwt/jwt/v5" // Uses JWT claims/signing/parsing primitives.
)

type JWTManager struct { // Holds JWT signing settings shared by login and middleware layers.
	secret []byte        // Secret key used for HS256 signing and verification.
	ttl    time.Duration // Token lifetime duration before expiration.
}

type Claims struct { // Custom JWT claims payload for authenticated user context.
	UserID               int64 `json:"user_id"` // Stores authenticated user ID in token payload.
	jwt.RegisteredClaims       // Embeds standard claims like exp and iat.
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager { // Constructor for reusable JWT manager dependency.
	return &JWTManager{
		secret: []byte(secret), // Converts secret string to bytes required by signing API.
		ttl:    ttl,            // Stores configured token lifetime.
	}
}

func (m *JWTManager) GenerateToken(userID int64) (string, error) { // Creates signed JWT string for a specific authenticated user.
	now := time.Now().UTC() // Captures current UTC time to set issued/expiry claims consistently.
	claims := Claims{
		UserID: userID, // Adds authenticated user identifier to custom claim.
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),            // Sets token issued-at timestamp.
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)), // Sets token expiration based on configured TTL.
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // Creates token using HS256 algorithm and populated claims.
	signed, err := token.SignedString(m.secret)                // Signs token with configured secret key.
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err) // Wraps sign failure with operation context.
	}
	return signed, nil // Returns signed bearer token string to caller.
}

func (m *JWTManager) ParseToken(token string) (int64, error) { // Validates token signature/expiry and extracts user ID claim.
	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) { // Parses token and checks claims structure.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok { // Rejects unexpected signing algorithms.
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil // Provides HMAC secret used to verify signature.
	})
	if err != nil {
		return 0, fmt.Errorf("parse jwt token: %w", err) // Wraps parse/validation failures.
	}

	claims, ok := parsedToken.Claims.(*Claims) // Casts parsed claims into expected custom claims type.
	if !ok || !parsedToken.Valid {             // Ensures claims type is correct and token fully valid.
		return 0, fmt.Errorf("invalid jwt token") // Returns invalid token error when cast/validation fails.
	}
	return claims.UserID, nil // Returns authenticated user ID claim.
}
