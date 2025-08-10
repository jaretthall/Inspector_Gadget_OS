package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestJWTManager_GenerateAndValidateToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:    "test-secret-key-12345",
		TokenExpiry:  time.Hour,
		Issuer:       "test-issuer",
		AllowedRoles: []string{"admin", "user"},
	}

	jwtManager := NewJWTManager(config)

	// Generate token
	userID := "user123"
	username := "testuser"
	roles := []string{"user", "admin"}

	tokenString, err := jwtManager.GenerateToken(userID, username, roles)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Validate token
	claims, err := jwtManager.ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, roles, claims.Roles)
	assert.Equal(t, config.Issuer, claims.Issuer)
}

func TestJWTManager_InvalidToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:   "test-secret-key-12345",
		TokenExpiry: time.Hour,
	}

	jwtManager := NewJWTManager(config)

	// Test invalid token
	_, err := jwtManager.ValidateToken("invalid.token.string")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)

	// Test empty token
	_, err = jwtManager.ValidateToken("")
	assert.Error(t, err)
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:   "test-secret-key-12345",
		TokenExpiry: -time.Hour, // Expired immediately
	}

	jwtManager := NewJWTManager(config)

	tokenString, err := jwtManager.GenerateToken("user123", "testuser", []string{"user"})
	assert.NoError(t, err)

	// Token should be expired
	_, err = jwtManager.ValidateToken(tokenString)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestJWTManager_CheckRole(t *testing.T) {
	config := JWTConfig{
		SecretKey:    "test-secret-key-12345",
		AllowedRoles: []string{"admin", "user", "viewer"},
	}

	jwtManager := NewJWTManager(config)

	testCases := []struct {
		name          string
		userRoles     []string
		requiredRoles []string
		expected      bool
	}{
		{
			name:          "User has required role",
			userRoles:     []string{"user", "viewer"},
			requiredRoles: []string{"user"},
			expected:      true,
		},
		{
			name:          "User has one of multiple required roles",
			userRoles:     []string{"viewer"},
			requiredRoles: []string{"admin", "viewer"},
			expected:      true,
		},
		{
			name:          "User doesn't have required role",
			userRoles:     []string{"viewer"},
			requiredRoles: []string{"admin"},
			expected:      false,
		},
		{
			name:          "No role requirements",
			userRoles:     []string{"viewer"},
			requiredRoles: []string{},
			expected:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := jwtManager.CheckRole(tc.userRoles, tc.requiredRoles)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJWTMiddleware(t *testing.T) {
	config := JWTConfig{
		SecretKey:    "test-secret-key-12345",
		TokenExpiry:  time.Hour,
		AllowedRoles: []string{"admin", "user"},
	}

	jwtManager := NewJWTManager(config)

	// Generate valid token
	tokenString, err := jwtManager.GenerateToken("user123", "testuser", []string{"user"})
	assert.NoError(t, err)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(jwtManager.Middleware())
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Test with valid token
	t.Run("Valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Test without token
	t.Run("Missing token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	// Test with invalid token
	t.Run("Invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestRequireRoleMiddleware(t *testing.T) {
	config := JWTConfig{
		SecretKey:    "test-secret-key-12345",
		TokenExpiry:  time.Hour,
		AllowedRoles: []string{"admin", "user"},
	}

	jwtManager := NewJWTManager(config)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(jwtManager.Middleware())
	
	// Route requiring admin role
	router.GET("/admin", jwtManager.RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	// Generate tokens with different roles
	userToken, err := jwtManager.GenerateToken("user123", "normaluser", []string{"user"})
	assert.NoError(t, err)

	adminToken, err := jwtManager.GenerateToken("admin123", "adminuser", []string{"admin"})
	assert.NoError(t, err)

	// Test user trying to access admin route
	t.Run("User role accessing admin route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+userToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	// Test admin accessing admin route
	t.Run("Admin role accessing admin route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRefreshToken(t *testing.T) {
	config := JWTConfig{
		SecretKey:   "test-secret-key-12345",
		TokenExpiry: time.Hour,
	}

	jwtManager := NewJWTManager(config)

	// Generate original token
	originalToken, err := jwtManager.GenerateToken("user123", "testuser", []string{"user"})
	assert.NoError(t, err)

	// Refresh token
	refreshedToken, err := jwtManager.RefreshToken(originalToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, refreshedToken)
	assert.NotEqual(t, originalToken, refreshedToken)

	// Validate refreshed token
	claims, err := jwtManager.ValidateToken(refreshedToken)
	assert.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, []string{"user"}, claims.Roles)
}

func TestGetUserFromContext(t *testing.T) {
	config := JWTConfig{
		SecretKey:   "test-secret-key-12345",
		TokenExpiry: time.Hour,
	}

	jwtManager := NewJWTManager(config)
	tokenString, err := jwtManager.GenerateToken("user123", "testuser", []string{"user"})
	assert.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(jwtManager.Middleware())
	router.GET("/test", func(c *gin.Context) {
		claims, err := GetUserFromContext(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"roles":    claims.Roles,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}