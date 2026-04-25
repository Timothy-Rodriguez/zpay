package pkg

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTService defines an interface for JWT operations.
type JWTService interface {
	GenerateToken(claims map[string]interface{}, expiration time.Duration) (string, error)
	ValidateToken(tokenString string) (map[string]interface{}, error)
}

// jwtService is an implementation of JWTService.
type jwtService struct {
	secret []byte // Secret key used for signing and verifying tokens
	// expiration    time.Duration     // Default expiration duration for tokens
	signingMethod jwt.SigningMethod // Signing method (e.g., HS256)
}

// NewJWTService creates a new JWTService instance.
func NewJWTService(secret []byte) JWTService {
	return &jwtService{
		secret:        secret,
		signingMethod: jwt.SigningMethodHS256,
	}
}

// GenerateToken creates a new JWT with the provided claims and signs it using the secret key.
func (s *jwtService) GenerateToken(claims map[string]interface{}, expiration time.Duration) (string, error) {
	// If expiration isn't provided in claims, set the default expiration
	if _, expExists := claims["exp"]; !expExists {
		claims["exp"] = time.Now().Add(expiration).Unix()
	}

	// Create a new token with the specified signing method
	token := jwt.NewWithClaims(s.signingMethod, jwt.MapClaims(claims))

	// Sign the token with the secret key
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

// ValidateToken parses and validates a JWT string, returning its claims if valid.
func (s *jwtService) ValidateToken(tokenString string) (map[string]interface{}, error) {
	// Parse the token with validation
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method matches what we expect
		if token.Method != s.signingMethod {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	// Extract claims as a map
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	// Verify expiration (handled by jwt.Parse, but explicit check for clarity)
	if exp, ok := claims["exp"].(float64); ok {
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			return nil, jwt.ErrTokenExpired
		}
	}

	return claims, nil
}
