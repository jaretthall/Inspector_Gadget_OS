// Package rbac provides RBAC middleware integration with JWT authentication
package rbac

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "inspector-gadget-os/o-llama/internal/auth"
    "inspector-gadget-os/o-llama/internal/logging"
)

// RBACMiddleware creates Gin middleware for role-based access control
type RBACMiddleware struct {
	casbinManager *CasbinManager
}

// NewRBACMiddleware creates a new RBAC middleware
func NewRBACMiddleware(casbinManager *CasbinManager) *RBACMiddleware {
	return &RBACMiddleware{
		casbinManager: casbinManager,
	}
}

// RequirePermission creates middleware that requires specific object:action permission
func (rm *RBACMiddleware) RequirePermission(object, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user claims from JWT middleware (must run after auth middleware)
		claims, err := auth.GetUserFromContext(c)
        if err != nil {
            logging.L().Warnw("rbac.denied", "route", c.FullPath(), "reason", "no auth")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Check direct user permission
		allowed, err := rm.casbinManager.CheckUserPermission(claims.Username, object, action)
		if err != nil {
            logging.L().Errorw("rbac.error", "route", c.FullPath(), "user", claims.Username, "object", object, "action", action, "error", err.Error())
            c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			c.Abort()
			return
		}

		if allowed {
			c.Next()
			return
		}

		// Check role-based permissions
		for _, role := range claims.Roles {
			roleKey := "role:" + role
			allowed, err := rm.casbinManager.Enforce(roleKey, object, action)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "role permission check failed"})
				c.Abort()
				return
			}
			if allowed {
				c.Next()
				return
			}
		}

        logging.L().Warnw("rbac.denied", "route", c.FullPath(), "user", claims.Username, "object", object, "action", action)
        c.JSON(http.StatusForbidden, gin.H{
			"error":   "insufficient permissions",
			"required": map[string]string{
				"object": object,
				"action": action,
			},
		})
		c.Abort()
	}
}

// RequireRole creates middleware that requires a specific role
func (rm *RBACMiddleware) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user claims from JWT middleware
		claims, err := auth.GetUserFromContext(c)
        if err != nil {
            logging.L().Warnw("rbac.denied", "route", c.FullPath(), "reason", "no auth")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Check if user has the required role
		hasRole, err := rm.casbinManager.CheckUserRole(claims.Username, requiredRole)
        if err != nil {
            logging.L().Errorw("rbac.error", "route", c.FullPath(), "user", claims.Username, "required_role", requiredRole, "error", err.Error())
            c.JSON(http.StatusInternalServerError, gin.H{"error": "role check failed"})
			c.Abort()
			return
		}

        if !hasRole {
            logging.L().Warnw("rbac.denied", "route", c.FullPath(), "user", claims.Username, "required_role", requiredRole)
            c.JSON(http.StatusForbidden, gin.H{
				"error":        "insufficient role",
				"required_role": requiredRole,
				"user_roles":   claims.Roles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole creates middleware that requires any of the specified roles
func (rm *RBACMiddleware) RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user claims from JWT middleware
		claims, err := auth.GetUserFromContext(c)
        if err != nil {
            logging.L().Warnw("rbac.denied", "route", c.FullPath(), "reason", "no auth")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		for _, requiredRole := range roles {
			hasRole, err := rm.casbinManager.CheckUserRole(claims.Username, requiredRole)
            if err != nil {
				continue // Try next role
			}
			if hasRole {
				c.Next()
				return
			}
		}

        logging.L().Warnw("rbac.denied", "route", c.FullPath(), "user", claims.Username, "required_roles", roles)
        c.JSON(http.StatusForbidden, gin.H{
			"error":         "insufficient role",
			"required_roles": roles,
			"user_roles":    claims.Roles,
		})
		c.Abort()
	}
}

// RequireAllRoles creates middleware that requires all of the specified roles
func (rm *RBACMiddleware) RequireAllRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user claims from JWT middleware
		claims, err := auth.GetUserFromContext(c)
        if err != nil {
            logging.L().Warnw("rbac.denied", "route", c.FullPath(), "reason", "no auth")
            c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Check if user has all required roles
		for _, requiredRole := range roles {
			hasRole, err := rm.casbinManager.CheckUserRole(claims.Username, requiredRole)
            if err != nil || !hasRole {
                logging.L().Warnw("rbac.denied", "route", c.FullPath(), "user", claims.Username, "missing_role", requiredRole)
                c.JSON(http.StatusForbidden, gin.H{
					"error":         "insufficient roles",
					"required_roles": roles,
					"user_roles":    claims.Roles,
					"missing_role":  requiredRole,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// CheckPermission is a helper function to check permissions in handlers
func (rm *RBACMiddleware) CheckPermission(c *gin.Context, object, action string) bool {
	claims, err := auth.GetUserFromContext(c)
	if err != nil {
		return false
	}

	// Check direct user permission
	allowed, err := rm.casbinManager.CheckUserPermission(claims.Username, object, action)
	if err == nil && allowed {
		return true
	}

	// Check role-based permissions
	for _, role := range claims.Roles {
		roleKey := "role:" + role
		allowed, err := rm.casbinManager.Enforce(roleKey, object, action)
		if err == nil && allowed {
			return true
		}
	}

	return false
}

// GetUserPermissions returns all permissions for the current user
func (rm *RBACMiddleware) GetUserPermissions(c *gin.Context) ([]Permission, error) {
	claims, err := auth.GetUserFromContext(c)
	if err != nil {
		return nil, err
	}

	var allPermissions []Permission

	// Get role-based permissions
	for _, role := range claims.Roles {
		rolePerms, err := rm.casbinManager.GetRolePermissions(role)
		if err != nil {
			continue
		}
		allPermissions = append(allPermissions, rolePerms...)
	}

	return allPermissions, nil
}

// AdminOnly is a convenience middleware for admin-only routes
func (rm *RBACMiddleware) AdminOnly() gin.HandlerFunc {
	return rm.RequireRole("admin")
}

// UserOrAdmin is a convenience middleware for user or admin access
func (rm *RBACMiddleware) UserOrAdmin() gin.HandlerFunc {
	return rm.RequireAnyRole("user", "admin")
}

// FileSystemRead requires filesystem read permission
func (rm *RBACMiddleware) FileSystemRead() gin.HandlerFunc {
	return rm.RequirePermission("filesystem", "read")
}

// FileSystemWrite requires filesystem write permission
func (rm *RBACMiddleware) FileSystemWrite() gin.HandlerFunc {
	return rm.RequirePermission("filesystem", "write")
}

// AIAccess requires AI access permission
func (rm *RBACMiddleware) AIAccess() gin.HandlerFunc {
	return rm.RequirePermission("ai", "access")
}

// SystemManage requires system management permission
func (rm *RBACMiddleware) SystemManage() gin.HandlerFunc {
	return rm.RequirePermission("system", "manage")
}