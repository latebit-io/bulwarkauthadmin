//go:build integration

package rbac

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/rbac"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	integration.TestMain(m)
}

func TestRbacHandler_CreateRole(t *testing.T) {
	tc := integration.NewTestContext(t)

	roleName := fmt.Sprintf("role_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Test role description",
	}

	// Create role
	resp, err := tc.Post("/rbac/roles", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Try creating same role - should fail with conflict
	resp, err = tc.Post("/rbac/roles", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestRbacHandler_ListRoles(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a role first
	roleName := fmt.Sprintf("listrole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role for list test",
	}
	resp, err := tc.Post("/rbac/roles", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// List roles
	resp, err = tc.Get("/rbac/roles")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var roles []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&roles)
	require.NoError(t, err)

	// Should have at least one role
	assert.Greater(t, len(roles), 0)
}

func TestRbacHandler_GetRole(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a role
	roleName := fmt.Sprintf("getrole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role for get test",
	}
	resp, err := tc.Post("/rbac/roles", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get the role using path parameter
	resp, err = tc.Get("/rbac/roles/" + roleName)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var role map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&role)
	require.NoError(t, err)

	assert.Equal(t, roleName, role["name"])
	assert.Equal(t, "Role for get test", role["description"])
}

func TestRbacHandler_UpdateRole(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a role
	roleName := fmt.Sprintf("updaterole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Original description",
	}
	resp, err := tc.Post("/rbac/roles", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Update the role (only description in body, name from path)
	updatePayload := rbac.UpdateRoleRequest{
		Description: "Updated description",
	}

	resp, err = tc.Put("/rbac/roles/"+roleName, updatePayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify the update
	resp, err = tc.Get("/rbac/roles/" + roleName)
	require.NoError(t, err)

	var role map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&role)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "Updated description", role["description"])
}

func TestRbacHandler_DeleteRole(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a role
	roleName := fmt.Sprintf("deleterole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role to delete",
	}
	resp, err := tc.Post("/rbac/roles", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Delete the role (name from path parameter)
	resp, err = tc.Delete("/rbac/roles/" + roleName)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Try to get the deleted role - should return 404
	resp, err = tc.Get("/rbac/roles/" + roleName)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRbacHandler_CreatePermission(t *testing.T) {
	tc := integration.NewTestContext(t)

	permissionName := fmt.Sprintf("perm_%d", time.Now().UnixNano())
	payload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "read",
	}

	// Create permission
	resp, err := tc.Post("/rbac/permissions", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Try creating same permission - should fail with conflict
	resp, err = tc.Post("/rbac/permissions", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestRbacHandler_ListPermissions(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a permission first
	permissionName := fmt.Sprintf("listperm_%d", time.Now().UnixNano())
	payload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "write",
	}
	resp, err := tc.Post("/rbac/permissions", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// List permissions
	resp, err = tc.Get("/rbac/permissions")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var permissions []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&permissions)
	require.NoError(t, err)

	// Should have at least one permission
	assert.Greater(t, len(permissions), 0)
}

func TestRbacHandler_DeletePermission(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a permission
	permissionName := fmt.Sprintf("deleteperm_%d", time.Now().UnixNano())
	payload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "delete",
	}
	resp, err := tc.Post("/rbac/permissions", payload)
	require.NoError(t, err)
	resp.Body.Close()

	permissionKey := fmt.Sprintf("%s:%s", permissionName, "delete")
	resp, err = tc.Delete("/rbac/permissions/" + permissionKey)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestRbacHandler_RolePermissionFlow(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a role
	roleName := fmt.Sprintf("flowrole_%d", time.Now().UnixNano())
	rolePayload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role for permission flow test",
	}
	resp, err := tc.Post("/rbac/roles", rolePayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Create a permission
	permissionName := fmt.Sprintf("flowperm_%d", time.Now().UnixNano())
	permPayload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "execute",
	}
	resp, err = tc.Post("/rbac/permissions", permPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get permission list to find the key
	resp, err = tc.Get("/rbac/permissions")
	require.NoError(t, err)

	var permissions []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&permissions)
	require.NoError(t, err)
	resp.Body.Close()

	// Find our permission's key
	var permissionKey string
	for _, perm := range permissions {
		if perm["name"] == permissionName && perm["action"] == "execute" {
			permissionKey = perm["key"].(string)
			break
		}
	}
	require.NotEmpty(t, permissionKey, "Permission key not found")

	// Add permission to role (only permissionKey in body, roleName from path)
	addPermPayload := rbac.AddPermissionRoleRequest{
		PermissionKey: permissionKey,
	}

	resp, err = tc.Put("/rbac/roles/"+roleName+"/permissions", addPermPayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify role has the permission
	resp, err = tc.Get("/rbac/roles/" + roleName)
	require.NoError(t, err)

	var role map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&role)
	require.NoError(t, err)
	resp.Body.Close()

	rolePermissions := role["permissionIds"].([]interface{})
	assert.Contains(t, rolePermissions, permissionKey)

	// Remove permission from role
	resp, err = tc.Delete("/rbac/roles/" + roleName + "/permissions/" + permissionKey)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify permission removed
	resp, err = tc.Get("/rbac/roles/" + roleName)
	require.NoError(t, err)

	var updatedRole map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&updatedRole)
	require.NoError(t, err)
	resp.Body.Close()

	updatedPermissions := updatedRole["permissionIds"].([]interface{})
	assert.NotContains(t, updatedPermissions, permissionKey)
}
