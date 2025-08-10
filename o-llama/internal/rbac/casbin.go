// Package rbac provides Role-Based Access Control using Casbin for O-LLaMA
package rbac

import (
	"fmt"
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// CasbinManager handles RBAC operations using Casbin
type CasbinManager struct {
	enforcer *casbin.Enforcer
	adapter  *gormadapter.Adapter
	db       *gorm.DB
}

// CasbinConfig holds configuration for Casbin RBAC
type CasbinConfig struct {
	DatabasePath   string // SQLite database path
	ModelConfig    string // Casbin model configuration
	AutoSave       bool   // Auto-save policy changes
	EnableLogging  bool   // Enable Casbin logging
}

// Permission represents a permission in the RBAC system
type Permission struct {
	Subject string `json:"subject"` // User or role
	Object  string `json:"object"`  // Resource
	Action  string `json:"action"`  // Operation
}

// Role represents a role with its permissions
type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// User represents a user with roles
type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// Default Casbin model configuration for O-LLaMA
const DefaultModelConfig = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

// Predefined roles and permissions for O-LLaMA
var (
	DefaultRoles = map[string][]string{
		"admin": {
			"filesystem:read",
			"filesystem:write",
			"filesystem:execute",
			"system:config",
			"system:manage",
			"ai:access",
			"ai:models",
			"users:manage",
			"roles:manage",
		},
		"user": {
			"filesystem:read",
			"ai:access",
		},
		"readonly": {
			"filesystem:read",
		},
		"ai_user": {
			"filesystem:read",
			"filesystem:write",
			"ai:access",
		},
	}

	DefaultPermissions = []Permission{
		// Admin permissions
		{"role:admin", "filesystem", "read"},
		{"role:admin", "filesystem", "write"},
		{"role:admin", "filesystem", "execute"},
		{"role:admin", "system", "config"},
		{"role:admin", "system", "manage"},
		{"role:admin", "ai", "access"},
		{"role:admin", "ai", "models"},
		{"role:admin", "users", "manage"},
		{"role:admin", "roles", "manage"},
		{"role:admin", "gadgets", "execute"},
		{"role:admin", "gadgets", "manage"},

		// Regular user permissions
		{"role:user", "filesystem", "read"},
		{"role:user", "ai", "access"},
		{"role:user", "gadgets", "execute"},

		// Readonly permissions
		{"role:readonly", "filesystem", "read"},

		// AI user permissions
		{"role:ai_user", "filesystem", "read"},
		{"role:ai_user", "filesystem", "write"},
		{"role:ai_user", "ai", "access"},
		{"role:ai_user", "gadgets", "execute"},
	}
)

// NewCasbinManager creates a new RBAC manager with Casbin
func NewCasbinManager(config CasbinConfig) (*CasbinManager, error) {
	// Set defaults
	if config.DatabasePath == "" {
		config.DatabasePath = "./rbac.db"
	}
	if config.ModelConfig == "" {
		config.ModelConfig = DefaultModelConfig
	}

	// Initialize database
	db, err := gorm.Open(sqlite.Open(config.DatabasePath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize Casbin adapter
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin adapter: %w", err)
	}

	// Create model from configuration
	model, err := model.NewModelFromString(config.ModelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin model: %w", err)
	}

	// Create enforcer
	enforcer, err := casbin.NewEnforcer(model, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	// Configure enforcer
	if config.EnableLogging {
		enforcer.EnableLog(true)
	}

	manager := &CasbinManager{
		enforcer: enforcer,
		adapter:  adapter,
		db:       db,
	}

	// Load existing policy
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	// Initialize default roles and permissions if empty
	if err := manager.initializeDefaults(); err != nil {
		return nil, fmt.Errorf("failed to initialize defaults: %w", err)
	}

	return manager, nil
}

// initializeDefaults sets up default roles and permissions
func (cm *CasbinManager) initializeDefaults() error {
	// Check if any policies exist
	policies := cm.enforcer.GetPolicy()
	if len(policies) > 0 {
		return nil // Already initialized
	}

	log.Println("Initializing default RBAC policies...")

	// Add default permissions
	for _, perm := range DefaultPermissions {
		_, err := cm.enforcer.AddPolicy(perm.Subject, perm.Object, perm.Action)
		if err != nil {
			return fmt.Errorf("failed to add default permission %v: %w", perm, err)
		}
	}

	// Save policies
	if err := cm.enforcer.SavePolicy(); err != nil {
		return fmt.Errorf("failed to save default policies: %w", err)
	}

	log.Printf("Initialized %d default permissions", len(DefaultPermissions))
	return nil
}

// Enforce checks if a user has permission to perform an action on a resource
func (cm *CasbinManager) Enforce(subject, object, action string) (bool, error) {
	allowed, err := cm.enforcer.Enforce(subject, object, action)
	if err != nil {
		return false, fmt.Errorf("failed to enforce policy: %w", err)
	}
	return allowed, nil
}

// AddPermission adds a permission policy
func (cm *CasbinManager) AddPermission(subject, object, action string) error {
	added, err := cm.enforcer.AddPolicy(subject, object, action)
	if err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}
	if !added {
		return fmt.Errorf("permission already exists: %s %s %s", subject, object, action)
	}
	return nil
}

// RemovePermission removes a permission policy
func (cm *CasbinManager) RemovePermission(subject, object, action string) error {
	removed, err := cm.enforcer.RemovePolicy(subject, object, action)
	if err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}
	if !removed {
		return fmt.Errorf("permission not found: %s %s %s", subject, object, action)
	}
	return nil
}

// AssignRole assigns a role to a user
func (cm *CasbinManager) AssignRole(user, role string) error {
	added, err := cm.enforcer.AddRoleForUser(user, fmt.Sprintf("role:%s", role))
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}
	if !added {
		return fmt.Errorf("role already assigned: user %s already has role %s", user, role)
	}
	return nil
}

// RemoveRole removes a role from a user
func (cm *CasbinManager) RemoveRole(user, role string) error {
	removed, err := cm.enforcer.DeleteRoleForUser(user, fmt.Sprintf("role:%s", role))
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}
	if !removed {
		return fmt.Errorf("role not found: user %s does not have role %s", user, role)
	}
	return nil
}

// GetUserRoles returns all roles assigned to a user
func (cm *CasbinManager) GetUserRoles(user string) ([]string, error) {
	roles, err := cm.enforcer.GetRolesForUser(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Remove "role:" prefix
	cleanRoles := make([]string, len(roles))
	for i, role := range roles {
		if len(role) > 5 && role[:5] == "role:" {
			cleanRoles[i] = role[5:]
		} else {
			cleanRoles[i] = role
		}
	}

	return cleanRoles, nil
}

// GetRolePermissions returns all permissions for a role
func (cm *CasbinManager) GetRolePermissions(role string) ([]Permission, error) {
	roleKey := fmt.Sprintf("role:%s", role)
	policies := cm.enforcer.GetFilteredPolicy(0, roleKey)

	permissions := make([]Permission, len(policies))
	for i, policy := range policies {
		if len(policy) >= 3 {
			permissions[i] = Permission{
				Subject: policy[0],
				Object:  policy[1],
				Action:  policy[2],
			}
		}
	}

	return permissions, nil
}

// GetAllUsers returns all users in the system
func (cm *CasbinManager) GetAllUsers() ([]User, error) {
	groupings := cm.enforcer.GetGroupingPolicy()
	userMap := make(map[string]*User)

	// Build user map from role assignments
	for _, grouping := range groupings {
		if len(grouping) >= 2 {
			username := grouping[0]
			role := grouping[1]

			// Remove "role:" prefix
			if len(role) > 5 && role[:5] == "role:" {
				role = role[5:]
			}

			if user, exists := userMap[username]; exists {
				user.Roles = append(user.Roles, role)
			} else {
				userMap[username] = &User{
					ID:       username,
					Username: username,
					Roles:    []string{role},
				}
			}
		}
	}

	// Convert map to slice
	users := make([]User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, *user)
	}

	return users, nil
}

// GetAllRoles returns all available roles
func (cm *CasbinManager) GetAllRoles() ([]Role, error) {
	roleMap := make(map[string]*Role)

	// Get all policies to extract roles
	policies := cm.enforcer.GetPolicy()
	for _, policy := range policies {
		if len(policy) >= 3 && len(policy[0]) > 5 && policy[0][:5] == "role:" {
			roleName := policy[0][5:] // Remove "role:" prefix
			permission := fmt.Sprintf("%s:%s", policy[1], policy[2])

			if role, exists := roleMap[roleName]; exists {
				role.Permissions = append(role.Permissions, permission)
			} else {
				description := ""
				if desc, ok := getRoleDescription(roleName); ok {
					description = desc
				}
				roleMap[roleName] = &Role{
					Name:        roleName,
					Description: description,
					Permissions: []string{permission},
				}
			}
		}
	}

	// Convert map to slice
	roles := make([]Role, 0, len(roleMap))
	for _, role := range roleMap {
		roles = append(roles, *role)
	}

	return roles, nil
}

// getRoleDescription returns a description for predefined roles
func getRoleDescription(roleName string) (string, bool) {
	descriptions := map[string]string{
		"admin":    "Full system administrator with all permissions",
		"user":     "Regular user with basic file and AI access",
		"readonly": "Read-only access to filesystem",
		"ai_user":  "User with AI and limited file system access",
	}
	desc, ok := descriptions[roleName]
	return desc, ok
}

// CheckUserPermission checks if a user has a specific permission
func (cm *CasbinManager) CheckUserPermission(user, object, action string) (bool, error) {
	return cm.Enforce(user, object, action)
}

// CheckUserRole checks if a user has a specific role
func (cm *CasbinManager) CheckUserRole(user, role string) (bool, error) {
	roles, err := cm.GetUserRoles(user)
	if err != nil {
		return false, err
	}

	for _, userRole := range roles {
		if userRole == role {
			return true, nil
		}
	}
	return false, nil
}

// RefreshPolicy reloads the policy from storage
func (cm *CasbinManager) RefreshPolicy() error {
	return cm.enforcer.LoadPolicy()
}

// SavePolicy saves the current policy to storage
func (cm *CasbinManager) SavePolicy() error {
	return cm.enforcer.SavePolicy()
}

// Close closes the database connection
func (cm *CasbinManager) Close() error {
	if db, err := cm.db.DB(); err == nil {
		return db.Close()
	}
	return nil
}

// GetPolicyStats returns statistics about the current policy
func (cm *CasbinManager) GetPolicyStats() map[string]int {
	return map[string]int{
		"policies":  len(cm.enforcer.GetPolicy()),
		"groupings": len(cm.enforcer.GetGroupingPolicy()),
		"roles":     len(cm.getRoleNames()),
		"users":     len(cm.getUserNames()),
	}
}

// getRoleNames returns unique role names
func (cm *CasbinManager) getRoleNames() []string {
	roleSet := make(map[string]bool)
	policies := cm.enforcer.GetPolicy()
	for _, policy := range policies {
		if len(policy) > 0 && len(policy[0]) > 5 && policy[0][:5] == "role:" {
			roleSet[policy[0][5:]] = true
		}
	}

	roles := make([]string, 0, len(roleSet))
	for role := range roleSet {
		roles = append(roles, role)
	}
	return roles
}

// getUserNames returns unique user names
func (cm *CasbinManager) getUserNames() []string {
	userSet := make(map[string]bool)
	groupings := cm.enforcer.GetGroupingPolicy()
	for _, grouping := range groupings {
		if len(grouping) > 0 {
			userSet[grouping[0]] = true
		}
	}

	users := make([]string, 0, len(userSet))
	for user := range userSet {
		users = append(users, user)
	}
	return users
}