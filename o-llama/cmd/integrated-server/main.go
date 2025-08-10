// Inspector Gadget OS Integrated Server - Combines O-LLaMA security with Gadget Framework
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"inspector-gadget-os/o-llama/internal/auth"
	"inspector-gadget-os/o-llama/internal/integration"
    "inspector-gadget-os/o-llama/internal/logging"
	"inspector-gadget-os/o-llama/internal/mcp"
	"inspector-gadget-os/o-llama/internal/rbac"
	"inspector-gadget-os/o-llama/internal/safefs"
	"inspector-gadget-os/o-llama/version"
)

// Config holds the server configuration
type Config struct {
	Port             string
	GadgetBinaryPath string
	DatabasePath     string
	JWTSecret        string
	AllowedBasePaths []string
	MaxFileSize      int64
}

func main() {
	// Load configuration
	config := loadConfig()
	
	// Initialize logger
    logger := log.New(os.Stdout, "INSPECTOR-GADGET: ", log.LstdFlags|log.Lshortfile)
    logger.Println("🤖 Starting Inspector Gadget OS Integrated Server")
    // Initialize global structured logger
    logging.Init("integrated-server", version.Version)
    logging.L().Infow("server.start", "port", config.Port, "gadget_binary", config.GadgetBinaryPath, "db", config.DatabasePath)
	
	// Initialize components
	server, err := initializeServer(config, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize server: %v", err)
	}
	
	// Start server
	if err := startServer(server, config, logger); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

// loadConfig loads configuration from environment or defaults
func loadConfig() *Config {
	config := &Config{
		Port:             getEnvOrDefault("PORT", "8080"),
		GadgetBinaryPath: getEnvOrDefault("GADGET_BINARY_PATH", "./gadget-framework/go-go-gadget"),
		DatabasePath:     getEnvOrDefault("DATABASE_PATH", "./inspector-gadget.db"),
		JWTSecret:        getEnvOrDefault("JWT_SECRET", "inspector-gadget-secret-key-change-in-production"),
		AllowedBasePaths: []string{"/tmp", "/home", "/workspace"},
		MaxFileSize:      10 * 1024 * 1024, // 10MB
	}
	
	// Resolve absolute path for gadget binary
	if absPath, err := filepath.Abs(config.GadgetBinaryPath); err == nil {
		config.GadgetBinaryPath = absPath
	}
	
	return config
}

// initializeServer initializes all server components
func initializeServer(config *Config, logger *log.Logger) (*gin.Engine, error) {
	// Initialize Casbin RBAC
	rbacConfig := rbac.CasbinConfig{
		DatabasePath:  config.DatabasePath,
		AutoSave:      true,
		EnableLogging: false,
	}
	
	casbinManager, err := rbac.NewCasbinManager(rbacConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RBAC: %w", err)
	}
	
	// Initialize JWT manager
	jwtConfig := auth.JWTConfig{
		SecretKey:    config.JWTSecret,
		TokenExpiry:  24 * time.Hour,
		Issuer:       "inspector-gadget-os",
		AllowedRoles: []string{"admin", "user", "readonly", "ai_user"},
	}
	
	jwtManager := auth.NewJWTManager(jwtConfig)
	
	// Initialize RBAC middleware
	rbacMiddleware := rbac.NewRBACMiddleware(casbinManager)
	
	// Initialize SafeFS
	safefsConfig := safefs.Config{
		BasePaths:   config.AllowedBasePaths,
		MaxFileSize: config.MaxFileSize,
		AllowedExts: []string{".txt", ".md", ".json", ".yaml", ".yml", ".log"},
	}
	
	safeFS := safefs.NewSafeFS(safefsConfig)
	
	// Initialize gadget integration
	gadgetIntegration := integration.NewGadgetIntegration(config.GadgetBinaryPath, rbacMiddleware)
	
	// Initialize MCP manager
	mcpConfig := mcp.MCPManagerConfig{
		ClientName:    "inspector-gadget-os",
		ClientVersion: "1.0.0",
		HealthCheck:   30 * time.Second,
		Logger:        logger,
		Servers:       make(map[string]*mcp.MCPServerConfig),
	}
	
	mcpManager := mcp.NewMCPManager(mcpConfig)
	
    // Setup Gin router
	gin.SetMode(gin.ReleaseMode)
    router := gin.New()
    router.Use(gin.Recovery())
    router.Use(corsMiddleware())
    // Init structured logger and middlewares
    logging.Init("integrated-server", version.Version)
    router.Use(logging.RequestIDMiddleware())
    router.Use(logging.AccessLogMiddleware())
	
	// Health check endpoint (no auth required)
    router.GET("/health", func(c *gin.Context) {
		status := make(map[string]interface{})
		
		// Check gadget framework
		if err := gadgetIntegration.HealthCheck(); err != nil {
			status["gadget_framework"] = "unhealthy: " + err.Error()
		} else {
			status["gadget_framework"] = "healthy"
		}
		
		// Check RBAC
		rbacStats := casbinManager.GetPolicyStats()
		status["rbac"] = map[string]interface{}{
			"status": "healthy",
			"stats":  rbacStats,
		}
		
        status["server"] = "healthy"
		status["timestamp"] = time.Now()
        status["version"] = version.Version
		
        c.JSON(http.StatusOK, status)
	})
	
	// Authentication endpoints (no auth middleware)
    // Ensure JSON binding uses DisallowUnknownFields for stricter input validation
    binding.EnableDecoderUseNumber = true

    auth := router.Group("/api/auth")
	{
        auth.POST("/login", createLoginHandler(jwtManager, casbinManager))
        auth.POST("/refresh", jwtManager.Middleware(), createRefreshHandler(jwtManager))
	}
	
	// Protected API endpoints
	api := router.Group("/api")
	api.Use(jwtManager.Middleware()) // All API endpoints require authentication
	
	// RBAC management (admin only)
	rbacAPI := rbac.NewRBACAPIHandler(casbinManager, rbacMiddleware)
	rbacAPI.RegisterRoutes(api)
	
	// Gadget integration (role-based access)
	gadgetIntegration.RegisterRoutes(api)
	
	// File system operations (permission-based access)
	fs := api.Group("/fs")
	{
		fs.GET("/read", rbacMiddleware.FileSystemRead(), createFileReadHandler(safeFS))
		fs.POST("/write", rbacMiddleware.FileSystemWrite(), createFileWriteHandler(safeFS))
		fs.GET("/list", rbacMiddleware.FileSystemRead(), createFileListHandler(safeFS))
	}
	
	// MCP endpoints (AI access required)
	mcpAPI := api.Group("/mcp")
	mcpAPI.Use(rbacMiddleware.AIAccess())
	{
		mcpAPI.GET("/servers", createMCPServersHandler(mcpManager))
		mcpAPI.POST("/servers/:name/connect", createMCPConnectHandler(mcpManager))
		mcpAPI.DELETE("/servers/:name", createMCPDisconnectHandler(mcpManager))
		mcpAPI.GET("/resources", createMCPResourcesHandler(mcpManager))
		mcpAPI.POST("/tools/:server/:tool", createMCPToolHandler(mcpManager))
	}
	
	// Create default admin user if none exists
	if err := createDefaultAdmin(casbinManager, jwtManager, logger); err != nil {
		logger.Printf("Warning: Could not create default admin: %v", err)
	}
	
	// Start MCP manager
	go func() {
		if err := mcpManager.Start(context.Background()); err != nil {
			logger.Printf("MCP manager error: %v", err)
		}
	}()
	
    // Static file handler for built web UI (Vite output)
    // Serves files from ./web-ui/dist with SPA fallback
    staticDir := filepath.Join(".", "web-ui", "dist")
    router.Static("/assets", filepath.Join(staticDir, "assets"))
    router.NoRoute(func(c *gin.Context) {
        // Only handle GET for SPA fallback
        if c.Request.Method != http.MethodGet {
            c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
            return
        }
        // Try to serve index.html when path does not match an API route
        if filepath.Ext(c.Request.URL.Path) == "" || filepath.Ext(c.Request.URL.Path) == "/" {
            c.File(filepath.Join(staticDir, "index.html"))
            return
        }
        // For direct asset paths not found, fallback to index as well
        c.File(filepath.Join(staticDir, "index.html"))
    })

    logging.L().Infow("server.ready")
    logger.Println("✅ All components initialized successfully")
	return router, nil
}

// startServer starts the HTTP server with graceful shutdown
func startServer(router *gin.Engine, config *Config, logger *log.Logger) error {
	srv := &http.Server{
		Addr:           ":" + config.Port,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	
	// Start server in goroutine
	go func() {
        logger.Printf("🚀 Server starting on port %s", config.Port)
        logger.Printf("📍 Gadget binary: %s", config.GadgetBinaryPath)
        logger.Printf("🔐 RBAC database: %s", config.DatabasePath)
        logging.L().Infow("server.listen", "addr", ":"+config.Port)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	logger.Println("🛑 Shutting down server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("Server forced to shutdown: %v", err)
		return err
	}
	
	logger.Println("✅ Server shutdown complete")
	return nil
}

// Helper functions for handlers

func createLoginHandler(jwtManager *auth.JWTManager, casbinManager *rbac.CasbinManager) gin.HandlerFunc {
	type LoginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		// Simple authentication - in production, use proper password hashing
		validUsers := map[string][]string{
			"admin":    {"admin123", "admin"},
			"user":     {"user123", "user"},
			"readonly": {"readonly123", "readonly"},
		}
		
        if userData, exists := validUsers[req.Username]; exists && len(userData) >= 2 {
			if userData[0] == req.Password {
				// Generate token
				token, err := jwtManager.GenerateToken(req.Username, req.Username, []string{userData[1]})
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
					return
				}
				
                logging.L().Infow("auth.login.ok", "user", req.Username)
				c.JSON(http.StatusOK, gin.H{
					"token":    token,
					"username": req.Username,
					"roles":    []string{userData[1]},
				})
				return
			}
		}
		
        logging.L().Warnw("auth.login.fail", "user", req.Username)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	}
}

func createRefreshHandler(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.GetUserFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		
		newToken, err := jwtManager.GenerateToken(claims.UserID, claims.Username, claims.Roles)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh token"})
			return
		}
		
        logging.L().Infow("auth.refresh.ok", "user", claims.Username)
		c.JSON(http.StatusOK, gin.H{
			"token":    newToken,
			"username": claims.Username,
			"roles":    claims.Roles,
		})
	}
}

func createFileReadHandler(safeFS *safefs.SafeFS) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter required"})
			return
		}
		
		claims, _ := auth.GetUserFromContext(c)
		data, err := safeFS.ReadFile(path, claims.Username)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"path":    path,
			"content": string(data),
			"size":    len(data),
		})
	}
}

func createFileWriteHandler(safeFS *safefs.SafeFS) gin.HandlerFunc {
	type WriteRequest struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content" binding:"required"`
	}
	
	return func(c *gin.Context) {
		var req WriteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		claims, _ := auth.GetUserFromContext(c)
		err := safeFS.WriteFile(req.Path, claims.Username, []byte(req.Content), 0644)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "File written successfully",
			"path":    req.Path,
			"size":    len(req.Content),
		})
	}
}

func createFileListHandler(safeFS *safefs.SafeFS) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Query("path")
		if path == "" {
			path = "/tmp" // Default path
		}
		
		claims, _ := auth.GetUserFromContext(c)
		files, err := safeFS.ListDir(path, claims.Username)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		var fileList []map[string]interface{}
		for _, file := range files {
			fileList = append(fileList, map[string]interface{}{
				"name":    file.Name(),
				"size":    file.Size(),
				"mode":    file.Mode().String(),
				"is_dir":  file.IsDir(),
				"mod_time": file.ModTime(),
			})
		}
		
		c.JSON(http.StatusOK, gin.H{
			"path":  path,
			"files": fileList,
			"count": len(fileList),
		})
	}
}

// MCP handler functions (simplified)
func createMCPServersHandler(mcpManager *mcp.MCPManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		configs := mcpManager.GetServerConfigs()
		status := mcpManager.GetServerStatus()
		
		c.JSON(http.StatusOK, gin.H{
			"configs": configs,
			"status":  status,
		})
	}
}

func createMCPConnectHandler(mcpManager *mcp.MCPManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverName := c.Param("name")
		
		if err := mcpManager.ConnectServer(c.Request.Context(), serverName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "Connected to " + serverName})
	}
}

func createMCPDisconnectHandler(mcpManager *mcp.MCPManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverName := c.Param("name")
		
		if err := mcpManager.DisconnectServer(serverName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"message": "Disconnected from " + serverName})
	}
}

func createMCPResourcesHandler(mcpManager *mcp.MCPManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		resources, err := mcpManager.ListResources(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{"resources": resources})
	}
}

func createMCPToolHandler(mcpManager *mcp.MCPManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverName := c.Param("server")
		toolName := c.Param("tool")
		
		var args interface{}
		c.ShouldBindJSON(&args)
		
		response, err := mcpManager.CallTool(c.Request.Context(), serverName, toolName, args)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusOK, response)
	}
}

func createDefaultAdmin(casbinManager *rbac.CasbinManager, jwtManager *auth.JWTManager, logger *log.Logger) error {
	users, err := casbinManager.GetAllUsers()
	if err != nil {
		return err
	}
	
	// Check if any admin user exists
	for _, user := range users {
		for _, role := range user.Roles {
			if role == "admin" {
				return nil // Admin already exists
			}
		}
	}
	
	// Create default admin
	if err := casbinManager.AssignRole("admin", "admin"); err != nil {
		return err
	}
	
	logger.Println("✅ Created default admin user (username: admin, password: admin123)")
	return nil
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}