package auth // Provides JWT creation and validation helpers for authenticated routes.

import (
	"fmt"  // Uses fmt.Errorf to wrap JWT parse/sign errors.
	"time" // Uses time.Now and time.Duration for token expiry handling.

	"github.com/golang-jwt/jwt/v5" // Uses JWT claims/signing/parsing primitives.
)

type JWTManager struct { // Holds JWT signing settings shared by login and middleware layers.
	secret     []byte        // Secret key used for HS256 signing and verification.
	accessTTL  time.Duration // Access token lifetime duration before expiration.
	refreshTTL time.Duration // Refresh token lifetime duration before expiration.
}

type Claims struct { // Custom JWT claims payload for authenticated user context.
	UserID               int64  `json:"user_id"`   // Stores authenticated user ID in token payload.
	TokenUse             string `json:"token_use"` // Distinguishes access tokens from refresh tokens.
	jwt.RegisteredClaims        // Embeds standard claims like exp and iat.
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager { // Constructor for reusable JWT manager dependency.
	return &JWTManager{
		secret:     []byte(secret), // Converts secret string to bytes required by signing API.
		accessTTL:  accessTTL,      // Stores configured access-token lifetime.
		refreshTTL: refreshTTL,     // Stores configured refresh-token lifetime.
	}
}

func (m *JWTManager) GenerateTokenPair(userID int64) (string, string, error) { // Creates access and refresh token pair for authenticated user.
	accessToken, err := m.generateToken(userID, "access", m.accessTTL) // Creates short-lived access token for API authorization.
	if err != nil {
		return "", "", err // Returns error when access token creation fails.
	}
	refreshToken, err := m.generateToken(userID, "refresh", m.refreshTTL) // Creates long-lived refresh token for token renewal.
	if err != nil {
		return "", "", err // Returns error when refresh token creation fails.
	}
	return accessToken, refreshToken, nil // Returns both tokens on success.
}

func (m *JWTManager) RefreshTokens(refreshToken string) (string, string, error) { // Validates refresh token and issues a new token pair.
	userID, err := m.parseTokenForUse(refreshToken, "refresh") // Ensures provided token is a valid refresh token.
	if err != nil {
		return "", "", err // Returns validation error for invalid/expired refresh token.
	}
	return m.GenerateTokenPair(userID) // Issues fresh access and refresh tokens for same user.
}

func (m *JWTManager) ParseAccessToken(token string) (int64, error) { // Validates access token and extracts user ID claim.
	return m.parseTokenForUse(token, "access") // Restricts accepted token type to access tokens only.
}

func (m *JWTManager) generateToken(userID int64, tokenUse string, ttl time.Duration) (string, error) { // Creates signed JWT string for a specific user and token use.
	now := time.Now().UTC() // Captures current UTC time to set issued/expiry claims consistently.
	claims := Claims{
		UserID:   userID,   // Adds authenticated user identifier to custom claim.
		TokenUse: tokenUse, // Marks token purpose as access or refresh.
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),          // Sets token issued-at timestamp.
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), // Sets token expiration based on configured TTL.
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // Creates token using HS256 algorithm and populated claims.
	signed, err := token.SignedString(m.secret)                // Signs token with configured secret key.
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err) // Wraps sign failure with operation context.
	}
	return signed, nil // Returns signed bearer token string to caller.
}

func (m *JWTManager) parseTokenForUse(token, expectedUse string) (int64, error) { // Validates token signature/expiry, enforces token use, and extracts user ID.
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
	if claims.TokenUse != expectedUse { // Verifies token purpose matches endpoint expectation.
		return 0, fmt.Errorf("invalid token type") // Returns error when access/refresh token is used in wrong place.
	}
	return claims.UserID, nil // Returns authenticated user ID claim.
}
