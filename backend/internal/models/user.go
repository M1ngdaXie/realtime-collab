package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
    ID             uuid.UUID  `json:"id"`
    Email          string     `json:"email"`
    Name           string     `json:"name"`
    AvatarURL      *string    `json:"avatar_url"`
    Provider       string     `json:"provider"`
    ProviderUserID string     `json:"-"` // Don't expose
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    LastLoginAt    *time.Time `json:"last_login_at"`
}

type Session struct {
    ID        uuid.UUID `json:"id"`
    UserID    uuid.UUID `json:"user_id"`
    TokenHash string    `json:"-"` // SHA-256 hash of JWT
    ExpiresAt time.Time `json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
    UserAgent *string   `json:"user_agent"`
    IPAddress *string   `json:"ip_address"`
}

type OAuthToken struct {
    ID           uuid.UUID  `json:"id"`
    UserID       uuid.UUID  `json:"user_id"`
    Provider     string     `json:"provider"`
    AccessToken  string     `json:"-"` // Don't expose
    RefreshToken *string    `json:"-"`
    TokenType    string     `json:"token_type"`
    ExpiresAt    time.Time  `json:"expires_at"`
    Scope        *string    `json:"scope"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}

// Response for /auth/me endpoint
type UserResponse struct {
    User  *User `json:"user"`
    Token string `json:"token,omitempty"`
}
