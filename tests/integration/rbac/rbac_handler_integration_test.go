package rbac

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/rbac"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Wait for service to be available
	integration.TestMain(m)
}

func TestRbacHandler_CreateRole(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()
	roleName := fmt.Sprintf("role_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Test role description",
	}
	body, _ := json.Marshal(payload)

	// Create role
	resp, err := http.Post(
		baseURL+"/api/v1/rbac/roles",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Try creating same role - should fail with conflict
	resp, err = http.Post(
		baseURL+"/api/v1/rbac/roles",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()
}

func TestRbacHandler_ListRoles(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create a role first
	roleName := fmt.Sprintf("listrole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role for list test",
	}
	body, _ := json.Marshal(payload)
	resp, _ := http.Post(baseURL+"/api/v1/rbac/roles", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// List roles
	resp, err := http.Get(baseURL + "/api/v1/rbac/roles")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var roles []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&roles)
	resp.Body.Close()

	// Should have at least one role
	assert.Greater(t, len(roles), 0)
}

func TestRbacHandler_GetRole(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create a role
	roleName := fmt.Sprintf("getrole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role for get test",
	}
	body, _ := json.Marshal(payload)
	resp, _ := http.Post(baseURL+"/api/v1/rbac/roles", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get the role using path parameter
	resp, err := http.Get(baseURL + "/api/v1/rbac/roles/" + roleName)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var role map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&role)
	resp.Body.Close()

	assert.Equal(t, roleName, role["name"])
	assert.Equal(t, "Role for get test", role["description"])
}

func TestRbacHandler_UpdateRole(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create a role
	roleName := fmt.Sprintf("updaterole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Original description",
	}
	body, _ := json.Marshal(payload)
	resp, _ := http.Post(baseURL+"/api/v1/rbac/roles", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Update the role (only description in body, name from path)
	updatePayload := rbac.UpdateRoleRequest{
		Description: "Updated description",
	}
	updateBody, _ := json.Marshal(updatePayload)

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/api/v1/rbac/roles/"+roleName, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify the update
	resp, _ = http.Get(baseURL + "/api/v1/rbac/roles/" + roleName)

	var role map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&role)
	resp.Body.Close()

	assert.Equal(t, "Updated description", role["description"])
}

func TestRbacHandler_DeleteRole(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create a role
	roleName := fmt.Sprintf("deleterole_%d", time.Now().UnixNano())
	payload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role to delete",
	}
	body, _ := json.Marshal(payload)
	resp, _ := http.Post(baseURL+"/api/v1/rbac/roles", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Delete the role (name from path parameter)
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/v1/rbac/roles/"+roleName, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Try to get the deleted role - should return 404
	resp, _ = http.Get(baseURL + "/api/v1/rbac/roles/" + roleName)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestRbacHandler_CreatePermission(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()
	permissionName := fmt.Sprintf("perm_%d", time.Now().UnixNano())
	payload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "read",
	}
	body, _ := json.Marshal(payload)

	// Create permission
	resp, err := http.Post(
		baseURL+"/api/v1/rbac/permissions",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Try creating same permission - should fail with conflict
	resp, err = http.Post(
		baseURL+"/api/v1/rbac/permissions",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()
}

func TestRbacHandler_ListPermissions(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create a permission first
	permissionName := fmt.Sprintf("listperm_%d", time.Now().UnixNano())
	payload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "write",
	}
	body, _ := json.Marshal(payload)
	resp, _ := http.Post(baseURL+"/api/v1/rbac/permissions", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// List permissions
	resp, err := http.Get(baseURL + "/api/v1/rbac/permissions")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var permissions []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&permissions)
	resp.Body.Close()

	// Should have at least one permission
	assert.Greater(t, len(permissions), 0)
}

func TestRbacHandler_DeletePermission(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create a permission
	permissionName := fmt.Sprintf("deleteperm_%d", time.Now().UnixNano())
	payload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "delete",
	}
	body, _ := json.Marshal(payload)
	resp, _ := http.Post(baseURL+"/api/v1/rbac/permissions", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Delete the permission
	deletePayload := rbac.DeletePermissionRequest{
		Name:   permissionName,
		Action: "delete",
	}
	deleteBody, _ := json.Marshal(deletePayload)

	permissionKey := fmt.Sprintf("%s:%s", permissionName, "delete")
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/v1/rbac/permissions/"+permissionKey, bytes.NewReader(deleteBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestRbacHandler_RolePermissionFlow(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create a role
	roleName := fmt.Sprintf("flowrole_%d", time.Now().UnixNano())
	rolePayload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "Role for permission flow test",
	}
	roleBody, _ := json.Marshal(rolePayload)
	resp, _ := http.Post(baseURL+"/api/v1/rbac/roles", "application/json", bytes.NewReader(roleBody))
	resp.Body.Close()

	// Create a permission
	permissionName := fmt.Sprintf("flowperm_%d", time.Now().UnixNano())
	permPayload := rbac.NewPermissionRequest{
		Name:   permissionName,
		Action: "execute",
	}
	permBody, _ := json.Marshal(permPayload)
	resp, _ = http.Post(baseURL+"/api/v1/rbac/permissions", "application/json", bytes.NewReader(permBody))
	resp.Body.Close()

	// Get permission list to find the key
	resp, _ = http.Get(baseURL + "/api/v1/rbac/permissions")
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var permissions []map[string]interface{}
	json.Unmarshal(respBody, &permissions)

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
	addPermBody, _ := json.Marshal(addPermPayload)

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/api/v1/rbac/roles/"+roleName+"/permissions", bytes.NewReader(addPermBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify role has the permission
	getResp, _ := http.Get(baseURL + "/api/v1/rbac/roles/" + roleName)
	var role map[string]interface{}
	json.NewDecoder(getResp.Body).Decode(&role)
	getResp.Body.Close()

	rolePermissions := role["permissionIds"].([]interface{})
	assert.Contains(t, rolePermissions, permissionKey)

	// Remove permission from role (only permissionKey in body, roleName from path)
	removePermPayload := rbac.RemovePermissionRoleRequest{
		PermissionKey: permissionKey,
	}
	removePermBody, _ := json.Marshal(removePermPayload)

	req, _ = http.NewRequest(http.MethodDelete, baseURL+"/api/v1/rbac/roles/"+roleName+"/permissions/"+removePermPayload.PermissionKey, bytes.NewReader(removePermBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify permission removed
	getResp2, _ := http.Get(baseURL + "/api/v1/rbac/roles/" + roleName)
	var updatedRole map[string]interface{}
	json.NewDecoder(getResp2.Body).Decode(&updatedRole)
	getResp2.Body.Close()

	updatedPermissions := updatedRole["permissionIds"].([]interface{})
	assert.NotContains(t, updatedPermissions, permissionKey)
}
