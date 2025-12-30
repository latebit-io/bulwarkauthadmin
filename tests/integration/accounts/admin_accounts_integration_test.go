//go:build integration

package accounts

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdminService_InternalRolesCreated verifies that the admin service
// CreateInternalRoles was called at startup and created the bulwark_admin role and permission
func TestAdminService_InternalRolesCreated(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Test 1: Verify bulwark_admin role exists
	resp, err := http.Get(baseURL + "/api/v1/rbac/roles")
	require.NoError(t, err)
	defer resp.Body.Close()

	var roles []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&roles)
	require.NoError(t, err)

	var bulwarkAdminRole map[string]interface{}
	for _, role := range roles {
		if role["name"] == "bulwark_admin" {
			bulwarkAdminRole = role
			break
		}
	}

	require.NotNil(t, bulwarkAdminRole, "bulwark_admin role should be created at startup by AdminAccountsService.CreateInternalRoles()")
	assert.Equal(t, "bulwark internal admin", bulwarkAdminRole["description"], "Role should have correct description")

	// Test 2: Verify role has the bulwark_admin:write permission
	permissions, ok := bulwarkAdminRole["permissionIds"].([]interface{})
	require.True(t, ok, "Role should have permissions array")
	assert.Contains(t, permissions, "bulwark_admin:write", "Role should contain bulwark_admin:write permission")

	// Test 3: Verify bulwark_admin:write permission exists
	resp, err = http.Get(baseURL + "/api/v1/rbac/permissions")
	require.NoError(t, err)
	defer resp.Body.Close()

	var perms []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&perms)
	require.NoError(t, err)

	var adminPermission map[string]interface{}
	for _, perm := range perms {
		if perm["key"] == "bulwark_admin:write" {
			adminPermission = perm
			break
		}
	}

	require.NotNil(t, adminPermission, "bulwark_admin:write permission should be created at startup")
	assert.Equal(t, "bulwark_admin", adminPermission["name"], "Permission name should be bulwark_admin")
	assert.Equal(t, "write", adminPermission["action"], "Permission action should be write")
}

// TestAdminService_RoleConstants verifies the admin role uses the expected constant values
func TestAdminService_RoleConstants(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	resp, err := http.Get(baseURL + "/api/v1/rbac/roles")
	require.NoError(t, err)
	defer resp.Body.Close()

	var roles []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&roles)
	require.NoError(t, err)

	for _, role := range roles {
		if role["name"] == "bulwark_admin" {
			// Verify exact values match constants from admin_accounts.go:
			// bulwarkAdminRole = "bulwark_admin"
			// bulwarkAdminRoleDescription = "bulwark internal admin"
			assert.Equal(t, "bulwark_admin", role["name"])
			assert.Equal(t, "bulwark internal admin", role["description"])
			return
		}
	}

	t.Fatal("bulwark_admin role not found")
}

// TestAdminService_PermissionConstants verifies the admin permission uses expected constant values
func TestAdminService_PermissionConstants(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	resp, err := http.Get(baseURL + "/api/v1/rbac/permissions")
	require.NoError(t, err)
	defer resp.Body.Close()

	var perms []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&perms)
	require.NoError(t, err)

	for _, perm := range perms {
		if perm["key"] == "bulwark_admin:write" {
			// Verify exact values match constants from admin_accounts.go:
			// bulwarkAdminPermission = "bulwark_admin"
			// bulwarkAdminAction = "write"
			assert.Equal(t, "bulwark_admin:write", perm["key"])
			assert.Equal(t, "bulwark_admin", perm["name"])
			assert.Equal(t, "write", perm["action"])
			return
		}
	}

	t.Fatal("bulwark_admin:write permission not found")
}

// TestAdminService_InternalRolesIdempotency verifies that CreateInternalRoles
// is safe to call multiple times (service restarts don't create duplicates)
func TestAdminService_InternalRolesIdempotency(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Count how many bulwark_admin roles exist (should be exactly 1)
	resp, err := http.Get(baseURL + "/api/v1/rbac/roles")
	require.NoError(t, err)
	defer resp.Body.Close()

	var roles []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&roles)
	require.NoError(t, err)

	adminRoleCount := 0
	for _, role := range roles {
		if role["name"] == "bulwark_admin" {
			adminRoleCount++
		}
	}

	assert.Equal(t, 1, adminRoleCount, "Should have exactly one bulwark_admin role (CreateInternalRoles should be idempotent)")

	// Count how many bulwark_admin:write permissions exist (should be exactly 1)
	resp, err = http.Get(baseURL + "/api/v1/rbac/permissions")
	require.NoError(t, err)
	defer resp.Body.Close()

	var perms []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&perms)
	require.NoError(t, err)

	adminPermCount := 0
	for _, perm := range perms {
		if perm["key"] == "bulwark_admin:write" {
			adminPermCount++
		}
	}

	assert.Equal(t, 1, adminPermCount, "Should have exactly one bulwark_admin:write permission")
}
