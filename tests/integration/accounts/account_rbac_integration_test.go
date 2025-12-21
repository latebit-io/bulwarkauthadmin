//go:build integration

package accounts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/accounts"
	rbacapi "github.com/latebit-io/bulwarkauthadmin/api/accounts/rbac"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRBACHandler_AssignRole(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account first
	email := fmt.Sprintf("rbac-assign-role%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID, "Account not found")

	// Assign role to account
	assignPayload := rbacapi.AssignRoleRequest{
		AccountID: accountID,
		Role:      "admin",
	}
	assignBody, _ := json.Marshal(assignPayload)

	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify role was assigned
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles := account["roles"].([]interface{})
	assert.Len(t, roles, 1)
	assert.Equal(t, "admin", roles[0])
}

func TestAccountRBACHandler_AssignRole_Idempotent(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-assign-role-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Assign role twice
	assignPayload := rbacapi.AssignRoleRequest{
		AccountID: accountID,
		Role:      "editor",
	}
	assignBody, _ := json.Marshal(assignPayload)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	resp.Body.Close()

	// Assign same role again
	req, _ = http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify role appears only once
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles := account["roles"].([]interface{})
	assert.Len(t, roles, 1, "Role should only appear once after duplicate assignment")
	assert.Equal(t, "editor", roles[0])
}

func TestAccountRBACHandler_AssignMultipleRoles(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-multiple-roles%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Assign multiple roles
	client := &http.Client{}
	roles := []string{"admin", "editor", "viewer"}
	for _, role := range roles {
		assignPayload := rbacapi.AssignRoleRequest{
			AccountID: accountID,
			Role:      role,
		}
		assignBody, _ := json.Marshal(assignPayload)

		req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role", bytes.NewReader(assignBody))
		req.Header.Set("Content-Type", "application/json")
		resp, _ = client.Do(req)
		resp.Body.Close()
	}

	// Verify all roles were assigned
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	accountRoles := account["roles"].([]interface{})
	assert.Len(t, accountRoles, 3)

	// Convert to string slice for easier comparison
	roleStrings := make([]string, len(accountRoles))
	for i, r := range accountRoles {
		roleStrings[i] = r.(string)
	}
	assert.Contains(t, roleStrings, "admin")
	assert.Contains(t, roleStrings, "editor")
	assert.Contains(t, roleStrings, "viewer")
}

func TestAccountRBACHandler_RemoveRole(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-remove-role%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Assign role first
	assignPayload := rbacapi.AssignRoleRequest{
		AccountID: accountID,
		Role:      "admin",
	}
	assignBody, _ := json.Marshal(assignPayload)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	resp.Body.Close()

	// Remove role
	req, _ = http.NewRequest(http.MethodDelete, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role/admin", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify role was removed
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles, ok := account["roles"].([]interface{})
	if ok {
		assert.Empty(t, roles, "Roles array should be empty after removal")
	}
}

func TestAccountRBACHandler_RemoveRole_Idempotent(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-remove-role-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Remove role that doesn't exist (should succeed without error)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role/nonexistent", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestAccountRBACHandler_AssignPermission(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-assign-permission%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Assign permission to account
	assignPayload := rbacapi.AssignPermissionRequest{
		AccountID:  accountID,
		Permission: "users:delete",
	}
	assignBody, _ := json.Marshal(assignPayload)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify permission was assigned
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	permissions := account["permissions"].([]interface{})
	assert.Len(t, permissions, 1)
	assert.Equal(t, "users:delete", permissions[0])
}

func TestAccountRBACHandler_AssignPermission_Idempotent(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-assign-permission-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Assign permission twice
	assignPayload := rbacapi.AssignPermissionRequest{
		AccountID:  accountID,
		Permission: "posts:create",
	}
	assignBody, _ := json.Marshal(assignPayload)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	resp.Body.Close()

	// Assign same permission again
	req, _ = http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify permission appears only once
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	permissions := account["permissions"].([]interface{})
	assert.Len(t, permissions, 1, "Permission should only appear once after duplicate assignment")
	assert.Equal(t, "posts:create", permissions[0])
}

func TestAccountRBACHandler_AssignMultiplePermissions(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-multiple-permissions%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Assign multiple permissions
	client := &http.Client{}
	permissions := []string{"users:create", "users:read", "users:delete"}
	for _, perm := range permissions {
		assignPayload := rbacapi.AssignPermissionRequest{
			AccountID:  accountID,
			Permission: perm,
		}
		assignBody, _ := json.Marshal(assignPayload)

		req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission", bytes.NewReader(assignBody))
		req.Header.Set("Content-Type", "application/json")
		resp, _ = client.Do(req)
		resp.Body.Close()
	}

	// Verify all permissions were assigned
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	accountPermissions := account["permissions"].([]interface{})
	assert.Len(t, accountPermissions, 3)

	// Convert to string slice for easier comparison
	permStrings := make([]string, len(accountPermissions))
	for i, p := range accountPermissions {
		permStrings[i] = p.(string)
	}
	assert.Contains(t, permStrings, "users:create")
	assert.Contains(t, permStrings, "users:read")
	assert.Contains(t, permStrings, "users:delete")
}

func TestAccountRBACHandler_RemovePermission(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-remove-permission%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Assign permission first
	assignPayload := rbacapi.AssignPermissionRequest{
		AccountID:  accountID,
		Permission: "comments:moderate",
	}
	assignBody, _ := json.Marshal(assignPayload)

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission", bytes.NewReader(assignBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	resp.Body.Close()

	// Remove permission
	req, _ = http.NewRequest(http.MethodDelete, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission/comments:moderate", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify permission was removed
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	permissions, ok := account["permissions"].([]interface{})
	if ok {
		assert.Empty(t, permissions, "Permissions array should be empty after removal")
	}
}

func TestAccountRBACHandler_RemovePermission_Idempotent(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-remove-permission-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Remove permission that doesn't exist (should succeed without error)
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission/nonexistent:action", nil)
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestAccountRBACHandler_RolesAndPermissionsTogether(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("rbac-combined%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/v1/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/v1/accounts")
	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	client := &http.Client{}

	// Assign roles
	rolePayload := rbacapi.AssignRoleRequest{
		AccountID: accountID,
		Role:      "editor",
	}
	roleBody, _ := json.Marshal(rolePayload)
	req, _ := http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/role", bytes.NewReader(roleBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	resp.Body.Close()

	// Assign permissions
	permPayload := rbacapi.AssignPermissionRequest{
		AccountID:  accountID,
		Permission: "special:feature",
	}
	permBody, _ := json.Marshal(permPayload)
	req, _ = http.NewRequest(http.MethodPatch, baseURL+"/api/v1/accounts/"+accountID+"/rbac/permission", bytes.NewReader(permBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	resp.Body.Close()

	// Verify both roles and permissions
	resp, _ = http.Get(baseURL + "/api/v1/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles := account["roles"].([]interface{})
	permissions := account["permissions"].([]interface{})

	assert.Len(t, roles, 1)
	assert.Equal(t, "editor", roles[0])
	assert.Len(t, permissions, 1)
	assert.Equal(t, "special:feature", permissions[0])
}
