//go:build integration

package accounts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/accounts"
	rbacapi "github.com/latebit-io/bulwarkauthadmin/api/accounts/rbac"
	"github.com/latebit-io/bulwarkauthadmin/api/rbac"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRBACHandler_AssignRole(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account first
	email := fmt.Sprintf("rbac-assign-role%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create "admin" role
	roleName := fmt.Sprintf("admin_%d", time.Now().UnixNano())
	rolePayload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "admin role",
	}

	resp, err = tc.Post("/rbac/roles", rolePayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Assign role to account
	assignPayload := rbacapi.AssignRoleRequest{
		Role: roleName,
	}

	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/role", assignPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify role was assigned
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles := account["roles"].([]interface{})
	assert.Len(t, roles, 1)
	assert.Equal(t, roleName, roles[0])
}

func TestAccountRBACHandler_AssignRole_Idempotent(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-assign-role-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create "editor" role
	roleName := fmt.Sprintf("editor_%d", time.Now().UnixNano())
	rolePayload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "editor role",
	}

	resp, err = tc.Post("/rbac/roles", rolePayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Assign role twice
	assignPayload := rbacapi.AssignRoleRequest{
		Role: roleName,
	}

	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/role", assignPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Assign same role again
	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/role", assignPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify role appears only once
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles := account["roles"].([]interface{})
	assert.Len(t, roles, 1, "Role should only appear once after duplicate assignment")
	assert.Equal(t, roleName, roles[0])
}

func TestAccountRBACHandler_AssignMultipleRoles(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-multiple-roles%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create roles with unique names
	timestamp := time.Now().UnixNano()
	roleNames := []string{
		fmt.Sprintf("admin_%d", timestamp),
		fmt.Sprintf("editor_%d", timestamp),
		fmt.Sprintf("viewer_%d", timestamp),
	}

	for _, roleName := range roleNames {
		rolePayload := rbac.NewRoleRequest{
			Name:        roleName,
			Description: roleName + " role",
		}

		resp, err = tc.Post("/rbac/roles", rolePayload)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Assign multiple roles
	for _, role := range roleNames {
		assignPayload := rbacapi.AssignRoleRequest{
			Role: role,
		}

		resp, err = tc.Patch("/accounts/"+accountID+"/rbac/role", assignPayload)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Verify all roles were assigned
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

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
	for _, roleName := range roleNames {
		assert.Contains(t, roleStrings, roleName)
	}
}

func TestAccountRBACHandler_RemoveRole(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-remove-role%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create role first
	roleName := fmt.Sprintf("admin_%d", time.Now().UnixNano())
	rolePayload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "admin role",
	}

	resp, err = tc.Post("/rbac/roles", rolePayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Assign role to account
	assignPayload := rbacapi.AssignRoleRequest{
		Role: roleName,
	}

	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/role", assignPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Remove role
	resp, err = tc.Delete("/accounts/" + accountID + "/rbac/role/" + roleName)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify role was removed
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles, ok := account["roles"].([]interface{})
	if ok {
		assert.Empty(t, roles, "Roles array should be empty after removal")
	}
}

func TestAccountRBACHandler_RemoveRole_Idempotent(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-remove-role-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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
	resp, err = tc.Delete("/accounts/" + accountID + "/rbac/role/nonexistent")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestAccountRBACHandler_AssignPermission(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-assign-permission%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create permission before adding to account
	permName := fmt.Sprintf("users_%d", time.Now().UnixNano())
	createPermissionPayload := rbac.NewPermissionRequest{
		Name:   permName,
		Action: "delete",
	}

	resp, err = tc.Post("/rbac/permissions", createPermissionPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Assign permission to account
	permissionKey := permName + ":delete"
	assignPayload := rbacapi.AssignPermissionRequest{
		Permission: permissionKey,
	}

	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/permission", assignPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify permission was assigned
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	permissions := account["permissions"].([]interface{})
	assert.Len(t, permissions, 1)
	assert.Equal(t, permissionKey, permissions[0])
}

func TestAccountRBACHandler_AssignPermission_Idempotent(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-assign-permission-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create permission before adding to account
	permName := fmt.Sprintf("posts_%d", time.Now().UnixNano())
	createPermissionPayload := rbac.NewPermissionRequest{
		Name:   permName,
		Action: "create",
	}

	resp, err = tc.Post("/rbac/permissions", createPermissionPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Assign permission twice
	permissionKey := permName + ":create"
	assignPayload := rbacapi.AssignPermissionRequest{
		Permission: permissionKey,
	}

	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/permission", assignPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Assign same permission again
	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/permission", assignPayload)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify permission appears only once
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	permissions := account["permissions"].([]interface{})
	assert.Len(t, permissions, 1, "Permission should only appear once after duplicate assignment")
	assert.Equal(t, permissionKey, permissions[0])
}

func TestAccountRBACHandler_AssignMultiplePermissions(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-multiple-permissions%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create permissions first with unique names
	timestamp := time.Now().UnixNano()
	permissionsToCreate := []struct {
		name   string
		action string
	}{
		{fmt.Sprintf("users_%d", timestamp), "create"},
		{fmt.Sprintf("users_%d", timestamp), "read"},
		{fmt.Sprintf("users_%d", timestamp), "delete"},
	}

	var permissionKeys []string
	for _, p := range permissionsToCreate {
		createPermPayload := rbac.NewPermissionRequest{
			Name:   p.name,
			Action: p.action,
		}

		resp, err = tc.Post("/rbac/permissions", createPermPayload)
		require.NoError(t, err)
		resp.Body.Close()

		permissionKeys = append(permissionKeys, p.name+":"+p.action)
	}

	// Assign multiple permissions
	for _, perm := range permissionKeys {
		assignPayload := rbacapi.AssignPermissionRequest{
			Permission: perm,
		}

		resp, err = tc.Patch("/accounts/"+accountID+"/rbac/permission", assignPayload)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Verify all permissions were assigned
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

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
	for _, permKey := range permissionKeys {
		assert.Contains(t, permStrings, permKey)
	}
}

func TestAccountRBACHandler_RemovePermission(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-remove-permission%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create permission first
	permName := fmt.Sprintf("comments_%d", time.Now().UnixNano())
	createPermPayload := rbac.NewPermissionRequest{
		Name:   permName,
		Action: "moderate",
	}

	resp, err = tc.Post("/rbac/permissions", createPermPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Assign permission to account
	permissionKey := permName + ":moderate"
	assignPayload := rbacapi.AssignPermissionRequest{
		Permission: permissionKey,
	}

	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/permission", assignPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Remove permission
	resp, err = tc.Delete("/accounts/" + accountID + "/rbac/permission/" + permissionKey)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify permission was removed
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	permissions, ok := account["permissions"].([]interface{})
	if ok {
		assert.Empty(t, permissions, "Permissions array should be empty after removal")
	}
}

func TestAccountRBACHandler_RemovePermission_Idempotent(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-remove-permission-idempotent%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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
	resp, err = tc.Delete("/accounts/" + accountID + "/rbac/permission/nonexistent:action")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestAccountRBACHandler_RolesAndPermissionsTogether(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("rbac-combined%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get account ID
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

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

	// Create "editor" role
	roleName := fmt.Sprintf("editor_%d", time.Now().UnixNano())
	createRolePayload := rbac.NewRoleRequest{
		Name:        roleName,
		Description: "editor role",
	}

	resp, err = tc.Post("/rbac/roles", createRolePayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Create "special:feature" permission
	permName := fmt.Sprintf("special_%d", time.Now().UnixNano())
	createPermPayload := rbac.NewPermissionRequest{
		Name:   permName,
		Action: "feature",
	}

	resp, err = tc.Post("/rbac/permissions", createPermPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Assign role to account
	rolePayload := rbacapi.AssignRoleRequest{
		Role: roleName,
	}
	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/role", rolePayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Assign permission to account
	permissionKey := permName + ":feature"
	permPayload := rbacapi.AssignPermissionRequest{
		Permission: permissionKey,
	}
	resp, err = tc.Patch("/accounts/"+accountID+"/rbac/permission", permPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Verify both roles and permissions
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	roles := account["roles"].([]interface{})
	permissions := account["permissions"].([]interface{})

	assert.Len(t, roles, 1)
	assert.Equal(t, roleName, roles[0])
	assert.Len(t, permissions, 1)
	assert.Equal(t, permissionKey, permissions[0])
}
