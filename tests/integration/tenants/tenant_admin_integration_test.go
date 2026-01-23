//go:build integration

package tenants

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/accounts"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTenantAdminMiddlewareRequiresAuth verifies that tenant-scoped endpoints require authentication
func TestTenantAdminMiddlewareRequiresAuth(t *testing.T) {
	// Make request without authentication
	tenantID := integration.GetSystemTenantID()
	url := fmt.Sprintf("%s/api/v1/tenant/%s/accounts", integration.GetBaseURL(), tenantID)

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 401 Unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestTenantAdminMiddlewareRequiresTenantAdminRole verifies that non-admins cannot access tenant-scoped endpoints
func TestTenantAdminMiddlewareRequiresTenantAdminRole(t *testing.T) {
	// Setup: Create a regular user (not admin)
	tc := integration.NewTestContext(t)

	// Create a second account (not an admin)
	regularUserEmail := fmt.Sprintf("regularuser_%d@example.com", time.Now().UnixNano())
	regularUserPayload := accounts.NewAccountRequest{Email: regularUserEmail}

	resp, err := tc.Post("/accounts", regularUserPayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Authenticate as the regular user
	regularUserToken, err := integration.AuthenticateAsUser(t, tc.TenantID, regularUserEmail)
	require.NoError(t, err)

	// Try to access tenant-scoped admin endpoint as regular user
	url := fmt.Sprintf("%s/api/v1/tenant/%s/accounts", integration.GetBaseURL(), tc.TenantID)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, url, regularUserToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 403 Forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestTenantAdminCanAccessTenantEndpoints verifies that tenant admins can access tenant-scoped endpoints
func TestTenantAdminCanAccessTenantEndpoints(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create a new tenant for this test
	adminTC := integration.SetupSystemAdminContext(t)
	tenantName := fmt.Sprintf("TenantAdminTest_%d", time.Now().UnixNano())
	tenantPayload := map[string]string{
		"Name":        tenantName,
		"Description": "Test tenant for tenant admin",
		"Domain":      "test.example.com",
	}

	adminBaseURL := fmt.Sprintf("%s/api/v1/admin", integration.GetBaseURL())
	resp, err := adminRequest("POST", "/tenants", adminTC, tenantPayload)
	require.NoError(t, err)

	var createResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	resp.Body.Close()
	require.NoError(t, err)

	newTenantID := createResp["id"].(string)

	// Create a user in the new tenant
	adminBaseURL = fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), newTenantID)
	tenantAdminEmail := fmt.Sprintf("tenantadmin_%d@example.com", time.Now().UnixNano())
	accountPayload := accounts.NewAccountRequest{Email: tenantAdminEmail}

	// Use system admin token to create account in new tenant
	accountURL := adminBaseURL + "/accounts"
	resp, err = integration.MakeAuthenticatedRequest(http.MethodPost, accountURL, adminTC.AccessToken, nil)
	require.NoError(t, err)

	// Marshal the payload properly
	body, _ := json.Marshal(accountPayload)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodPost, accountURL, adminTC.AccessToken, body)
	require.NoError(t, err)

	var accountResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&accountResp)
	resp.Body.Close()
	require.NoError(t, err)

	tenantAdminAccountID := accountResp["id"].(string)

	// Assign tenant_admin role to the user
	roleURL := adminBaseURL + "/accounts/rbac/roles"
	assignPayload := map[string]string{
		"accountId": tenantAdminAccountID,
		"role":      "tenant_admin",
	}

	body, _ = json.Marshal(assignPayload)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodPost, roleURL, adminTC.AccessToken, body)
	require.NoError(t, err)
	resp.Body.Close()

	// Authenticate as the tenant admin user
	tenantAdminToken, err := integration.AuthenticateAsUser(t, newTenantID, tenantAdminEmail)
	require.NoError(t, err)

	// Now the tenant admin should be able to access their tenant's endpoints
	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, adminBaseURL+"/accounts", tenantAdminToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestTenantAdminCannotAccessOtherTenants verifies tenant isolation
func TestTenantAdminCannotAccessOtherTenants(t *testing.T) {
	// Setup: Create two tenants with admins
	adminTC := integration.SetupSystemAdminContext(t)

	// Create first tenant
	tenant1Name := fmt.Sprintf("Tenant1_%d", time.Now().UnixNano())
	tenant1Payload := map[string]string{
		"Name":        tenant1Name,
		"Description": "First test tenant",
		"Domain":      "tenant1.example.com",
	}

	adminBaseURL := fmt.Sprintf("%s/api/v1/admin", integration.GetBaseURL())
	resp, err := adminRequest("POST", "/tenants", adminTC, tenant1Payload)
	require.NoError(t, err)

	var tenant1Resp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenant1Resp)
	resp.Body.Close()
	require.NoError(t, err)

	tenant1ID := tenant1Resp["id"].(string)

	// Create second tenant
	tenant2Name := fmt.Sprintf("Tenant2_%d", time.Now().UnixNano())
	tenant2Payload := map[string]string{
		"Name":        tenant2Name,
		"Description": "Second test tenant",
		"Domain":      "tenant2.example.com",
	}

	resp, err = adminRequest("POST", "/tenants", adminTC, tenant2Payload)
	require.NoError(t, err)

	var tenant2Resp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenant2Resp)
	resp.Body.Close()
	require.NoError(t, err)

	tenant2ID := tenant2Resp["id"].(string)

	// Create and setup admin in tenant 1
	tenant1BaseURL := fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), tenant1ID)
	tenant1AdminEmail := fmt.Sprintf("admin1_%d@example.com", time.Now().UnixNano())
	accountPayload := accounts.NewAccountRequest{Email: tenant1AdminEmail}

	body, _ := json.Marshal(accountPayload)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodPost, tenant1BaseURL+"/accounts", adminTC.AccessToken, body)
	require.NoError(t, err)

	var accountResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&accountResp)
	resp.Body.Close()
	require.NoError(t, err)

	tenant1AdminID := accountResp["id"].(string)

	// Assign tenant_admin role in tenant 1
	rolePayload := map[string]string{
		"accountId": tenant1AdminID,
		"role":      "tenant_admin",
	}

	body, _ = json.Marshal(rolePayload)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodPost, tenant1BaseURL+"/accounts/rbac/roles", adminTC.AccessToken, body)
	require.NoError(t, err)
	resp.Body.Close()

	// Authenticate as tenant 1 admin
	tenant1AdminToken, err := integration.AuthenticateAsUser(t, tenant1ID, tenant1AdminEmail)
	require.NoError(t, err)

	// Tenant 1 admin tries to access tenant 2 - should fail
	tenant2BaseURL := fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), tenant2ID)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, tenant2BaseURL+"/accounts", tenant1AdminToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 403 Forbidden (either from tenant check or from the JWT validation)
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest,
		"Expected 403 or 400, got %d", resp.StatusCode)
}

// TestSystemAdminCanAccessAnyTenant verifies that system admins bypass tenant admin checks
func TestSystemAdminCanAccessAnyTenant(t *testing.T) {
	// Setup: Create a tenant
	adminTC := integration.SetupSystemAdminContext(t)

	// Create a new tenant
	tenantName := fmt.Sprintf("SystemAdminTest_%d", time.Now().UnixNano())
	tenantPayload := map[string]string{
		"Name":        tenantName,
		"Description": "Test tenant for system admin access",
		"Domain":      "sysadmin-test.example.com",
	}

	adminBaseURL := fmt.Sprintf("%s/api/v1/admin", integration.GetBaseURL())
	resp, err := adminRequest("POST", "/tenants", adminTC, tenantPayload)
	require.NoError(t, err)

	var tenantResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenantResp)
	resp.Body.Close()
	require.NoError(t, err)

	newTenantID := tenantResp["id"].(string)

	// System admin should be able to access the tenant's endpoints
	tenantBaseURL := fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), newTenantID)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, tenantBaseURL+"/accounts", adminTC.AccessToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 200 OK (system admin has implicit tenant admin access)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestTenantAdminRoleCreatedAutomatically verifies that tenant_admin role is created for new tenants
func TestTenantAdminRoleCreatedAutomatically(t *testing.T) {
	// Setup: Create a new tenant
	adminTC := integration.SetupSystemAdminContext(t)

	tenantName := fmt.Sprintf("RoleTest_%d", time.Now().UnixNano())
	tenantPayload := map[string]string{
		"Name":        tenantName,
		"Description": "Test tenant for role creation",
		"Domain":      "roletest.example.com",
	}

	adminBaseURL := fmt.Sprintf("%s/api/v1/admin", integration.GetBaseURL())
	resp, err := adminRequest("POST", "/tenants", adminTC, tenantPayload)
	require.NoError(t, err)

	var tenantResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenantResp)
	resp.Body.Close()
	require.NoError(t, err)

	newTenantID := tenantResp["id"].(string)

	// Check that the tenant_admin role exists
	tenantBaseURL := fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), newTenantID)
	rolesURL := tenantBaseURL + "/rbac/roles"

	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, rolesURL, adminTC.AccessToken, nil)
	require.NoError(t, err)

	var rolesResp []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rolesResp)
	resp.Body.Close()
	require.NoError(t, err)

	// Find the tenant_admin role
	foundTenantAdminRole := false
	for _, role := range rolesResp {
		if name, ok := role["name"].(string); ok && name == "tenant_admin" {
			foundTenantAdminRole = true
			break
		}
	}

	assert.True(t, foundTenantAdminRole, "tenant_admin role should be created automatically for new tenants")
}
