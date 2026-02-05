package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserClaims defines the custom claims structure
// Senior Tip: Embed information that helps you avoid DB lookups later.
type UserClaims struct {
	Name      string  `json:"user_name"`
	Email     string  `json:"user_email"`
	AvatarURL *string `json:"avatar_url"` // Nullable avatar URL to avoid DB queries
	jwt.RegisteredClaims
}

// GenerateJWT creates a stateless JWT token for a user
// Changed: Input is now userID (and optional role), not sessionID
func GenerateJWT(userID uuid.UUID, name string, email string, avatarURL *string, secret string, expiresIn time.Duration) (string, error) {
	claims := UserClaims{
		Name:      name,
		Email:     email,
		AvatarURL: avatarURL,
		RegisteredClaims: jwt.RegisteredClaims{
			// Standard claim "Subject" is technically where UserID belongs,
			// but having a typed UserID field is easier for Go type assertions.
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "realtime-collab", // Your app name
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT parses the token and extracts the UserClaims
// Changed: Returns *UserClaims so you can access UserID and Role directly
func ValidateJWT(tokenString, secret string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Security Check: Always validate the signing algorithm
		// to prevent "None" algorithm attacks.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	// Type assertion to get our custom struct back
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}
