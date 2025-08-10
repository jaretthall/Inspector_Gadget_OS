package rbac

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCasbinManager(t *testing.T) {
	// Use temporary database for testing
	dbPath := "./test_rbac.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	require.NotNil(t, manager)

	defer manager.Close()

	// Check that default policies were initialized
	stats := manager.GetPolicyStats()
	assert.Greater(t, stats["policies"], 0)
}

func TestBasicPermissions(t *testing.T) {
	dbPath := "./test_rbac_basic.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Test admin role permissions
	allowed, err := manager.Enforce("role:admin", "filesystem", "read")
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = manager.Enforce("role:admin", "system", "manage")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Test user role permissions
	allowed, err = manager.Enforce("role:user", "filesystem", "read")
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = manager.Enforce("role:user", "system", "manage")
	assert.NoError(t, err)
	assert.False(t, allowed) // Users shouldn't be able to manage system

	// Test readonly role permissions
	allowed, err = manager.Enforce("role:readonly", "filesystem", "read")
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = manager.Enforce("role:readonly", "filesystem", "write")
	assert.NoError(t, err)
	assert.False(t, allowed) // Readonly users can't write
}

func TestUserRoleManagement(t *testing.T) {
	dbPath := "./test_rbac_users.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Assign role to user
	err = manager.AssignRole("alice", "admin")
	assert.NoError(t, err)

	err = manager.AssignRole("bob", "user")
	assert.NoError(t, err)

	// Check user roles
	aliceRoles, err := manager.GetUserRoles("alice")
	assert.NoError(t, err)
	assert.Contains(t, aliceRoles, "admin")

	bobRoles, err := manager.GetUserRoles("bob")
	assert.NoError(t, err)
	assert.Contains(t, bobRoles, "user")

	// Test user permissions through roles
	allowed, err := manager.Enforce("alice", "system", "manage")
	assert.NoError(t, err)
	assert.True(t, allowed) // Alice has admin role

	allowed, err = manager.Enforce("bob", "system", "manage")
	assert.NoError(t, err)
	assert.False(t, allowed) // Bob only has user role

	// Remove role
	err = manager.RemoveRole("alice", "admin")
	assert.NoError(t, err)

	aliceRoles, err = manager.GetUserRoles("alice")
	assert.NoError(t, err)
	assert.NotContains(t, aliceRoles, "admin")
}

func TestPermissionManagement(t *testing.T) {
	dbPath := "./test_rbac_perms.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Add custom permission
	err = manager.AddPermission("role:custom", "api", "access")
	assert.NoError(t, err)

	// Test the new permission
	allowed, err := manager.Enforce("role:custom", "api", "access")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Remove permission
	err = manager.RemovePermission("role:custom", "api", "access")
	assert.NoError(t, err)

	// Should no longer be allowed
	allowed, err = manager.Enforce("role:custom", "api", "access")
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRolePermissions(t *testing.T) {
	dbPath := "./test_rbac_role_perms.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Get admin role permissions
	adminPerms, err := manager.GetRolePermissions("admin")
	assert.NoError(t, err)
	assert.Greater(t, len(adminPerms), 0)

	// Check that admin has filesystem:read permission
	hasFilesystemRead := false
	for _, perm := range adminPerms {
		if perm.Object == "filesystem" && perm.Action == "read" {
			hasFilesystemRead = true
			break
		}
	}
	assert.True(t, hasFilesystemRead)

	// Get user role permissions
	userPerms, err := manager.GetRolePermissions("user")
	assert.NoError(t, err)
	assert.Greater(t, len(userPerms), 0)
	assert.Less(t, len(userPerms), len(adminPerms)) // User should have fewer permissions than admin
}

func TestGetAllUsers(t *testing.T) {
	dbPath := "./test_rbac_all_users.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Add some users
	err = manager.AssignRole("alice", "admin")
	assert.NoError(t, err)

	err = manager.AssignRole("bob", "user")
	assert.NoError(t, err)

	err = manager.AssignRole("charlie", "readonly")
	assert.NoError(t, err)

	// Get all users
	users, err := manager.GetAllUsers()
	assert.NoError(t, err)
	assert.Len(t, users, 3)

	// Check user details
	userMap := make(map[string]User)
	for _, user := range users {
		userMap[user.Username] = user
	}

	alice, exists := userMap["alice"]
	assert.True(t, exists)
	assert.Contains(t, alice.Roles, "admin")

	bob, exists := userMap["bob"]
	assert.True(t, exists)
	assert.Contains(t, bob.Roles, "user")
}

func TestGetAllRoles(t *testing.T) {
	dbPath := "./test_rbac_all_roles.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Get all roles
	roles, err := manager.GetAllRoles()
	assert.NoError(t, err)
	assert.Greater(t, len(roles), 0)

	// Check that default roles exist
	roleNames := make([]string, len(roles))
	for i, role := range roles {
		roleNames[i] = role.Name
	}

	assert.Contains(t, roleNames, "admin")
	assert.Contains(t, roleNames, "user")
	assert.Contains(t, roleNames, "readonly")
	assert.Contains(t, roleNames, "ai_user")
}

func TestCheckUserPermission(t *testing.T) {
	dbPath := "./test_rbac_user_perm.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Assign roles to users
	err = manager.AssignRole("admin_user", "admin")
	assert.NoError(t, err)

	err = manager.AssignRole("regular_user", "user")
	assert.NoError(t, err)

	// Test admin user permissions
	allowed, err := manager.CheckUserPermission("admin_user", "system", "manage")
	assert.NoError(t, err)
	assert.True(t, allowed)

	// Test regular user permissions
	allowed, err = manager.CheckUserPermission("regular_user", "filesystem", "read")
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = manager.CheckUserPermission("regular_user", "system", "manage")
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCheckUserRole(t *testing.T) {
	dbPath := "./test_rbac_user_role.db"
	defer os.Remove(dbPath)

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	manager, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager.Close()

	// Assign role to user
	err = manager.AssignRole("test_user", "admin")
	assert.NoError(t, err)

	// Check user has admin role
	hasRole, err := manager.CheckUserRole("test_user", "admin")
	assert.NoError(t, err)
	assert.True(t, hasRole)

	// Check user doesn't have user role
	hasRole, err = manager.CheckUserRole("test_user", "user")
	assert.NoError(t, err)
	assert.False(t, hasRole)
}

func TestPolicyPersistence(t *testing.T) {
	dbPath := "./test_rbac_persistence.db"
	defer func() {
		os.Remove(dbPath)
	}()

	config := CasbinConfig{
		DatabasePath:  dbPath,
		AutoSave:      true,
		EnableLogging: false,
	}

	// Create manager and add custom policy
	manager1, err := NewCasbinManager(config)
	require.NoError(t, err)

	err = manager1.AssignRole("persistent_user", "admin")
	assert.NoError(t, err)

	err = manager1.AddPermission("role:test", "resource", "action")
	assert.NoError(t, err)

	manager1.Close()

	// Create new manager with same database
	manager2, err := NewCasbinManager(config)
	require.NoError(t, err)
	defer manager2.Close()

	// Check that policies persisted
	roles, err := manager2.GetUserRoles("persistent_user")
	assert.NoError(t, err)
	assert.Contains(t, roles, "admin")

	allowed, err := manager2.Enforce("role:test", "resource", "action")
	assert.NoError(t, err)
	assert.True(t, allowed)
}