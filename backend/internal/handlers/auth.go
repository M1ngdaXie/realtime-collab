package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/M1ngdaXie/realtime-collab/internal/auth"
	"github.com/M1ngdaXie/realtime-collab/internal/models"
	"github.com/M1ngdaXie/realtime-collab/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	store          store.Store
	googleConfig   *oauth2.Config
	githubConfig   *oauth2.Config
	jwtSecret      string
	frontendURL    string
}

func NewAuthHandler(store store.Store, jwtSecret, frontendURL string) *AuthHandler {
	googleConfig := auth.GetGoogleOAuthConfig(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
	)

	githubConfig := auth.GetGitHubOAuthConfig(
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		os.Getenv("GITHUB_REDIRECT_URL"),
	)

	return &AuthHandler{
		store:        store,
		googleConfig: googleConfig,
		githubConfig: githubConfig,
		jwtSecret:    jwtSecret,
		frontendURL:  frontendURL,
	}
}

// GoogleLogin redirects to Google OAuth
func (h *AuthHandler) GoogleLogin(c *gin.Context) {
    // Generate random state and set cookie
    oauthState := generateStateOauthCookie(c.Writer)
    url := h.googleConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
    c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles Google OAuth callback
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	oauthState, err := c.Cookie("oauthstate")
    if err != nil || c.Query("state") != oauthState {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid oauth state"})
        return
    }
	log.Println("Google callback state:", c.Query("state"))
	// Exchange code for token
	token, err := h.googleConfig.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from Google
	client := h.googleConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	log.Println("Google user info response status:", resp.Status)
	log.Println("Google user info response headers:", resp.Header)
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	json.Unmarshal(data, &userInfo)
	log.Println("Google user info:", userInfo)
	// Upsert user in database
	user, err := h.store.UpsertUser(
		c.Request.Context(),
		"google",
		userInfo.ID,
		userInfo.Email,
		userInfo.Name,
		&userInfo.Picture,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Create session and JWT
	jwt, err := h.createSessionAndJWT(c, user)
	if err != nil {
		fmt.Printf("❌ DATABASE ERROR: %v\n", err) 
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": fmt.Sprintf("CreateSession Error: %v", err), 
        })
		return
	}

	// Redirect to frontend with token
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", h.frontendURL, jwt)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// GithubLogin redirects to GitHub OAuth
func (h *AuthHandler) GithubLogin(c *gin.Context) {
	url := h.githubConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GithubCallback handles GitHub OAuth callback
func (h *AuthHandler) GithubCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No code provided"})
		return
	}

	// Exchange code for token
	token, err := h.githubConfig.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from GitHub
	client := h.githubConfig.Client(context.Background(), token)
	
	// Get user profile
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	var userInfo struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	json.Unmarshal(data, &userInfo)

	// If email is not public, fetch it separately
	if userInfo.Email == "" {
		emailResp, _ := client.Get("https://api.github.com/user/emails")
		if emailResp != nil {
			defer emailResp.Body.Close()
			emailData, _ := io.ReadAll(emailResp.Body)
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			json.Unmarshal(emailData, &emails)
			for _, e := range emails {
				if e.Primary {
					userInfo.Email = e.Email
					break
				}
			}
		}
	}

	// Use login as name if name is empty
	if userInfo.Name == "" {
		userInfo.Name = userInfo.Login
	}

	// Upsert user in database
	user, err := h.store.UpsertUser(
		c.Request.Context(),
		"github",
		fmt.Sprintf("%d", userInfo.ID),
		userInfo.Email,
		userInfo.Name,
		&userInfo.AvatarURL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Create session and JWT
	jwt, err := h.createSessionAndJWT(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Redirect to frontend with token
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", h.frontendURL, jwt)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// Me returns current user info
func (h *AuthHandler) Me(c *gin.Context) {
	userID := auth.GetUserFromContext(c)
	if userID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), *userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, models.UserResponse{User: user})
}

// Logout invalidates the session
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusOK, gin.H{"message": "Already logged out"})
		return
	}

	// Extract token
	token := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	if token != "" {
		h.store.DeleteSession(c.Request.Context(), token)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Helper: create session and JWT
func (h *AuthHandler) createSessionAndJWT(c *gin.Context, user *models.User) (string, error) {
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days

	// Generate JWT first (we need it for session) - now includes avatar URL
	jwt, err := auth.GenerateJWT(user.ID, user.Name, user.Email, user.AvatarURL, h.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return "", err
	}

	// Create session in database
	sessionID := uuid.New()
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()
	_, err = h.store.CreateSession(
		c.Request.Context(),
		user.ID,
		sessionID,
		jwt,
		expiresAt,
		&userAgent,
		&ipAddress,
	)
	if err != nil {
		return "", err
	}

	return jwt, nil
}
func generateStateOauthCookie(w http.ResponseWriter) string {
    b := make([]byte, 16)
n, err := rand.Read(b)
if err != nil || n != 16 {
    fmt.Printf("Failed to generate random state: %v\n", err)
    return ""  // Critical for CSRF security
}
    state := base64.URLEncoding.EncodeToString(b)

    cookie := http.Cookie{
        Name:     "oauthstate",
        Value:    state,
        Expires:  time.Now().Add(10 * time.Minute),
        HttpOnly: true,        // Prevents JavaScript access (XSS protection)
        Secure:   false,       // Must be false for http://localhost (set true in production)
        SameSite: http.SameSiteLaxMode,  // ✅ Allows same-site OAuth redirects
        Path:     "/",         // ✅ Ensures cookie is sent to all backend paths
    }
    http.SetCookie(w, &cookie)

    return state
}
