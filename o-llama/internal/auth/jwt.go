// Package auth provides JWT-based authentication middleware for O-LLaMA
package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
    "inspector-gadget-os/o-llama/internal/logging"
)

// JWTManager handles JWT token creation and validation
type JWTManager struct {
	secretKey     []byte
	tokenExpiry   time.Duration
	issuer        string
	allowedRoles  map[string]bool
}

// Claims represents JWT claims for O-LLaMA authentication
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTConfig holds configuration for JWT authentication
type JWTConfig struct {
	SecretKey     string
	TokenExpiry   time.Duration
	Issuer        string
	AllowedRoles  []string
}

// Common errors
var (
	ErrInvalidToken   = errors.New("invalid token")
	ErrTokenExpired   = errors.New("token expired")
	ErrMissingToken   = errors.New("missing authorization token")
	ErrInsufficientRole = errors.New("insufficient role permissions")
)

// NewJWTManager creates a new JWT manager with the given configuration
func NewJWTManager(config JWTConfig) *JWTManager {
	allowedRoles := make(map[string]bool)
	for _, role := range config.AllowedRoles {
		allowedRoles[role] = true
	}

	if config.TokenExpiry == 0 {
		config.TokenExpiry = 24 * time.Hour // Default 24 hours
	}

	if config.Issuer == "" {
		config.Issuer = "inspector-gadget-os"
	}

	return &JWTManager{
		secretKey:     []byte(config.SecretKey),
		tokenExpiry:   config.TokenExpiry,
		issuer:        config.Issuer,
		allowedRoles:  allowedRoles,
	}
}

// GenerateToken creates a new JWT token for a user
func (j *JWTManager) GenerateToken(userID, username string, roles []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    j.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates and parses a JWT token
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// CheckRole verifies if the user has any of the required roles
func (j *JWTManager) CheckRole(userRoles []string, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true // No role requirement
	}

	for _, userRole := range userRoles {
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}

	return false
}

// Middleware creates a Gin middleware for JWT authentication
func (j *JWTManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := j.extractToken(c.Request)
		if tokenString == "" {
            logging.L().Warnw("auth.missing", "route", c.FullPath())
			c.JSON(http.StatusUnauthorized, gin.H{"error": ErrMissingToken.Error()})
			c.Abort()
			return
		}

        claims, err := j.ValidateToken(tokenString)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, ErrTokenExpired) {
				status = http.StatusUnauthorized // Could be 401 for expired tokens
			}
            logging.L().Warnw("auth.invalid", "route", c.FullPath(), "error", err.Error())
			c.JSON(status, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Store claims in context for use by handlers
        c.Set("user_claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_roles", claims.Roles)

		c.Next()
	}
}

// RequireRole creates a middleware that requires specific roles
func (j *JWTManager) RequireRole(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("user_claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		userClaims, ok := claims.(*Claims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user claims"})
			c.Abort()
			return
		}

		if !j.CheckRole(userClaims.Roles, requiredRoles) {
			c.JSON(http.StatusForbidden, gin.H{"error": ErrInsufficientRole.Error()})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractToken extracts the JWT token from the request
func (j *JWTManager) extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Bearer token format: "Bearer <token>"
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Check query parameter as fallback
	return r.URL.Query().Get("token")
}

// RefreshToken creates a new token with extended expiry for an existing valid token
func (j *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Create new token with same claims but updated expiry
	return j.GenerateToken(claims.UserID, claims.Username, claims.Roles)
}

// GetUserFromContext extracts user information from Gin context
func GetUserFromContext(c *gin.Context) (*Claims, error) {
	claims, exists := c.Get("user_claims")
	if !exists {
		return nil, errors.New("user claims not found in context")
	}

	userClaims, ok := claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid user claims in context")
	}

	return userClaims, nil
}