package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const UserContextKey contextKey = "user"
const ContextUserIDKey = "user_id"

// AuthMiddleware provides auth middleware
type AuthMiddleware struct {
	store     store.Store
	jwtSecret string
	logger    *zap.Logger
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(store store.Store, jwtSecret string, logger *zap.Logger) *AuthMiddleware {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AuthMiddleware{
		store:     store,
		jwtSecret: jwtSecret,
		logger:    logger,
	}
}

// RequireAuth middleware requires valid authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, claims, err := m.getUserFromToken(c)

		if err != nil || user == nil {
			if err != nil {
				m.logger.Warn("auth failed",
					zap.Error(err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.FullPath()),
				)
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Note: Name and Email might be empty for old JWT tokens
		c.Set(ContextUserIDKey, user)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		if claims.AvatarURL != nil {
			c.Set("avatar_url", *claims.AvatarURL)
		}
		c.Next()
	}
}

// OptionalAuth middleware sets user if authenticated, but doesn't require it
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, claims, _ := m.getUserFromToken(c)
		if user != nil {
			c.Set(string(UserContextKey), user)
			c.Set(ContextUserIDKey, user)
			if claims != nil {
				c.Set("user_email", claims.Email)
				c.Set("user_name", claims.Name)
				if claims.AvatarURL != nil {
					c.Set("avatar_url", *claims.AvatarURL)
				}
			}
		}
		c.Next()
	}
}

// getUserFromToken parses the JWT and returns the UserID and the full Claims (for name/email)
// 注意：返回值变了，现在返回 (*uuid.UUID, *UserClaims, error)
func (m *AuthMiddleware) getUserFromToken(c *gin.Context) (*uuid.UUID, *UserClaims, error) {
	authHeader := c.GetHeader("Authorization")

	if authHeader == "" {
		return nil, nil, nil
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, nil, nil
	}

	tokenString := parts[1]

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 必须要验证签名算法是 HMAC (HS256)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return nil, nil, err
	}

	// 2. 验证 Token 有效性并提取 Claims
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		// 3. 把 String 类型的 Subject 转回 UUID
		// 因为我们在 GenerateJWT 里存的是 claims.Subject = userID.String()
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid user ID in token")
		}

		// 成功！直接返回 UUID 和 claims (里面包含 Name 和 Email)
		// 这一步完全没有查数据库，速度极快
		return &userID, claims, nil
	}

	return nil, nil, fmt.Errorf("invalid token claims")
}

// GetUserFromContext extracts user ID from context
func GetUserFromContext(c *gin.Context) *uuid.UUID {
	// 修正点：使用和存入时完全一样的 Key
	val, exists := c.Get(ContextUserIDKey)
	if !exists {
		return nil
	}

	// 修正点：断言为 *uuid.UUID (因为我们在中间件里存的就是这个类型)
	uid, ok := val.(*uuid.UUID)
	if !ok {
		return nil
	}

	return uid
}

// ValidateToken validates a JWT token and returns user ID, name, and avatar URL from JWT claims
// STATELESS: No database lookup - relies entirely on JWT signature and expiration
func (m *AuthMiddleware) ValidateToken(tokenString string) (*uuid.UUID, string, string, error) {
	// Parse and validate JWT signature and expiration
	claims, err := ValidateJWT(tokenString, m.jwtSecret)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid token: %w", err)
	}

	// Parse user ID from claims
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid user ID in token: %w", err)
	}

	// Extract avatar URL from claims (handle nil gracefully)
	avatarURL := ""
	if claims.AvatarURL != nil {
		avatarURL = *claims.AvatarURL
	}

	// Return user data from JWT claims - ZERO database queries!
	return &userID, claims.Name, avatarURL, nil
}
