// Package rbac provides REST API endpoints for RBAC management
package rbac

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"inspector-gadget-os/o-llama/internal/auth"
    "inspector-gadget-os/o-llama/internal/logging"
)

// RBACAPIHandler provides REST API endpoints for RBAC operations
type RBACAPIHandler struct {
	casbinManager *CasbinManager
	rbacMiddleware *RBACMiddleware
}

// NewRBACAPIHandler creates a new RBAC API handler
func NewRBACAPIHandler(casbinManager *CasbinManager, rbacMiddleware *RBACMiddleware) *RBACAPIHandler {
	return &RBACAPIHandler{
		casbinManager:  casbinManager,
		rbacMiddleware: rbacMiddleware,
	}
}

// RegisterRoutes registers RBAC API routes
func (h *RBACAPIHandler) RegisterRoutes(router *gin.RouterGroup) {
	rbac := router.Group("/rbac")
	
	// User management (admin only)
	users := rbac.Group("/users")
	users.Use(h.rbacMiddleware.AdminOnly())
	{
		users.GET("", h.GetAllUsers)
		users.GET("/:username", h.GetUser)
		users.POST("/:username/roles", h.AssignUserRole)
		users.DELETE("/:username/roles/:role", h.RemoveUserRole)
		users.GET("/:username/permissions", h.GetUserPermissions)
	}

	// Role management (admin only)
	roles := rbac.Group("/roles")
	roles.Use(h.rbacMiddleware.AdminOnly())
	{
		roles.GET("", h.GetAllRoles)
		roles.GET("/:role", h.GetRole)
		roles.POST("/:role/permissions", h.AddRolePermission)
		roles.DELETE("/:role/permissions", h.RemoveRolePermission)
	}

	// Permission management (admin only)
	permissions := rbac.Group("/permissions")
	permissions.Use(h.rbacMiddleware.AdminOnly())
	{
		permissions.POST("", h.AddPermission)
		permissions.DELETE("", h.RemovePermission)
	}

	// Current user info (authenticated users)
	rbac.GET("/me", h.rbacMiddleware.UserOrAdmin(), h.GetCurrentUser)
	rbac.GET("/me/permissions", h.rbacMiddleware.UserOrAdmin(), h.GetCurrentUserPermissions)
	
	// System info (admin only)
	rbac.GET("/stats", h.rbacMiddleware.AdminOnly(), h.GetStats)
}

// Request/Response structures
type AssignRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

type AddPermissionRequest struct {
	Subject string `json:"subject" binding:"required"`
	Object  string `json:"object" binding:"required"`
	Action  string `json:"action" binding:"required"`
}

type RemovePermissionRequest struct {
	Subject string `json:"subject" binding:"required"`
	Object  string `json:"object" binding:"required"`
	Action  string `json:"action" binding:"required"`
}

type UserResponse struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// GetAllUsers returns all users in the system
func (h *RBACAPIHandler) GetAllUsers(c *gin.Context) {
	users, err := h.casbinManager.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"count": len(users),
	})
}

// GetUser returns information about a specific user
func (h *RBACAPIHandler) GetUser(c *gin.Context) {
	username := c.Param("username")

	roles, err := h.casbinManager.GetUserRoles(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user roles"})
		return
	}

	// Get user permissions through roles
	var allPermissions []string
	for _, role := range roles {
		rolePerms, err := h.casbinManager.GetRolePermissions(role)
		if err != nil {
			continue
		}
		for _, perm := range rolePerms {
			permStr := perm.Object + ":" + perm.Action
			allPermissions = append(allPermissions, permStr)
		}
	}

	user := UserResponse{
		ID:          username,
		Username:    username,
		Roles:       roles,
		Permissions: allPermissions,
	}

	c.JSON(http.StatusOK, user)
}

// AssignUserRole assigns a role to a user
func (h *RBACAPIHandler) AssignUserRole(c *gin.Context) {
	username := c.Param("username")
	
	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

    if err := h.casbinManager.AssignRole(username, req.Role); err != nil {
        logging.L().Errorw("rbac.assign.error", "actor", c.GetString("username"), "target", username, "role", req.Role, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

    logging.L().Infow("rbac.assign.ok", "actor", c.GetString("username"), "target", username, "role", req.Role)
	c.JSON(http.StatusOK, gin.H{
		"message": "Role assigned successfully",
		"user":    username,
		"role":    req.Role,
	})
}

// RemoveUserRole removes a role from a user
func (h *RBACAPIHandler) RemoveUserRole(c *gin.Context) {
	username := c.Param("username")
	role := c.Param("role")

    if err := h.casbinManager.RemoveRole(username, role); err != nil {
        logging.L().Errorw("rbac.remove.error", "actor", c.GetString("username"), "target", username, "role", role, "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

    logging.L().Infow("rbac.remove.ok", "actor", c.GetString("username"), "target", username, "role", role)
	c.JSON(http.StatusOK, gin.H{
		"message": "Role removed successfully",
		"user":    username,
		"role":    role,
	})
}

// GetUserPermissions returns all permissions for a user
func (h *RBACAPIHandler) GetUserPermissions(c *gin.Context) {
	username := c.Param("username")

	roles, err := h.casbinManager.GetUserRoles(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user roles"})
		return
	}

	var allPermissions []Permission
	for _, role := range roles {
		rolePerms, err := h.casbinManager.GetRolePermissions(role)
		if err != nil {
			continue
		}
		allPermissions = append(allPermissions, rolePerms...)
	}

	c.JSON(http.StatusOK, gin.H{
		"user":        username,
		"permissions": allPermissions,
		"count":       len(allPermissions),
	})
}

// GetAllRoles returns all roles in the system
func (h *RBACAPIHandler) GetAllRoles(c *gin.Context) {
	roles, err := h.casbinManager.GetAllRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get roles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// GetRole returns information about a specific role
func (h *RBACAPIHandler) GetRole(c *gin.Context) {
	roleName := c.Param("role")

	permissions, err := h.casbinManager.GetRolePermissions(roleName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role permissions"})
		return
	}

	role := Role{
		Name:        roleName,
		Permissions: make([]string, len(permissions)),
	}

	for i, perm := range permissions {
		role.Permissions[i] = perm.Object + ":" + perm.Action
	}

	if desc, ok := getRoleDescription(roleName); ok {
		role.Description = desc
	}

	c.JSON(http.StatusOK, role)
}

// AddRolePermission adds a permission to a role
func (h *RBACAPIHandler) AddRolePermission(c *gin.Context) {
	roleName := c.Param("role")
	
	var req AddPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleKey := "role:" + roleName
	if err := h.casbinManager.AddPermission(roleKey, req.Object, req.Action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permission added to role successfully",
		"role":    roleName,
		"object":  req.Object,
		"action":  req.Action,
	})
}

// RemoveRolePermission removes a permission from a role
func (h *RBACAPIHandler) RemoveRolePermission(c *gin.Context) {
	roleName := c.Param("role")
	
	var req RemovePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	roleKey := "role:" + roleName
	if err := h.casbinManager.RemovePermission(roleKey, req.Object, req.Action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permission removed from role successfully",
		"role":    roleName,
		"object":  req.Object,
		"action":  req.Action,
	})
}

// AddPermission adds a new permission to the system
func (h *RBACAPIHandler) AddPermission(c *gin.Context) {
	var req AddPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.casbinManager.AddPermission(req.Subject, req.Object, req.Action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permission added successfully",
		"subject": req.Subject,
		"object":  req.Object,
		"action":  req.Action,
	})
}

// RemovePermission removes a permission from the system
func (h *RBACAPIHandler) RemovePermission(c *gin.Context) {
	var req RemovePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.casbinManager.RemovePermission(req.Subject, req.Object, req.Action); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Permission removed successfully",
		"subject": req.Subject,
		"object":  req.Object,
		"action":  req.Action,
	})
}

// GetCurrentUser returns information about the currently authenticated user
func (h *RBACAPIHandler) GetCurrentUser(c *gin.Context) {
	claims, err := auth.GetUserFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	roles, err := h.casbinManager.GetUserRoles(claims.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user roles"})
		return
	}

	user := UserResponse{
		ID:       claims.UserID,
		Username: claims.Username,
		Roles:    roles,
	}

	c.JSON(http.StatusOK, user)
}

// GetCurrentUserPermissions returns permissions for the currently authenticated user
func (h *RBACAPIHandler) GetCurrentUserPermissions(c *gin.Context) {
	permissions, err := h.rbacMiddleware.GetUserPermissions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user permissions"})
		return
	}

	claims, _ := auth.GetUserFromContext(c)

	c.JSON(http.StatusOK, gin.H{
		"user":        claims.Username,
		"permissions": permissions,
		"count":       len(permissions),
	})
}

// GetStats returns RBAC system statistics
func (h *RBACAPIHandler) GetStats(c *gin.Context) {
	stats := h.casbinManager.GetPolicyStats()
	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// HealthCheck provides a health check endpoint for RBAC system
func (h *RBACAPIHandler) HealthCheck(c *gin.Context) {
	// Test basic RBAC functionality
	allowed, err := h.casbinManager.Enforce("role:admin", "system", "manage")
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "RBAC system error: " + err.Error(),
		})
		return
	}

	if !allowed {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "RBAC permissions not working correctly",
		})
		return
	}

	stats := h.casbinManager.GetPolicyStats()
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"stats":  stats,
	})
}