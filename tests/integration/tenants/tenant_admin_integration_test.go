//go:build integration

package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwark-auth-guard"
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
	// Setup: Create a regular user (not admin) in bulwarkauth
	tc := integration.NewTestContext(t)

	// Create a regular user in bulwarkauth first (so we can authenticate)
	regularUserEmail := fmt.Sprintf("regularuser_%d@example.com", time.Now().UnixNano())
	regularUserPassword := "TestPassword123!"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Register user in bulwarkauth
	httpClient := &http.Client{}
	bulwarkGuard := bulwark.NewGuard(integration.GetBulwarkAuthURL(), httpClient)

	err := bulwarkGuard.Account.Create(ctx, tc.TenantID, regularUserEmail, regularUserPassword)
	require.NoError(t, err)

	// Verify the account using the verification helper
	verificationToken, err := getVerificationTokenFromEmail(regularUserEmail)
	require.NoError(t, err)

	err = bulwarkGuard.Account.Verify(ctx, tc.TenantID, regularUserEmail, verificationToken)
	require.NoError(t, err)

	// Authenticate as the regular user (this also creates an account in bulwarkauthadmin)
	regularUserToken, err := integration.AuthenticateAsUser(t, tc.TenantID, regularUserEmail)
	require.NoError(t, err)

	// Try to access tenant-scoped admin endpoint as regular user
	url := fmt.Sprintf("%s/api/v1/tenant/%s/accounts", integration.GetBaseURL(), tc.TenantID)
	resp, err := integration.MakeAuthenticatedRequest(http.MethodGet, url, regularUserToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 403 Forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestTenantAdminCanAccessTenantEndpoints verifies that tenant admins can access tenant-scoped endpoints
func TestTenantAdminCanAccessTenantEndpoints(t *testing.T) {
	// Create a new tenant for this test
	adminTC := integration.SetupSystemAdminContext(t)
	tenantName := fmt.Sprintf("TenantAdminTest_%d", time.Now().UnixNano())
	tenantPayload := map[string]string{
		"Name":        tenantName,
		"Description": "Test tenant for tenant admin",
		"Domain":      "test.example.com",
	}

	resp, err := adminRequest("POST", "/tenants", adminTC, tenantPayload)
	require.NoError(t, err)
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("tenant creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Get the newly created tenant ID
	newTenantID := getTenantIDByName(adminTC, tenantName)
	require.NotEmpty(t, newTenantID, "newly created tenant not found")

	tenantAdminEmail := fmt.Sprintf("tenantadmin_%d@example.com", time.Now().UnixNano())
	tenantAdminPassword := "TestPassword123!"

	// Create and verify the user in bulwarkauth first (with the new tenant ID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpClient := &http.Client{}
	bulwarkGuard := bulwark.NewGuard(integration.GetBulwarkAuthURL(), httpClient)

	err = bulwarkGuard.Account.Create(ctx, newTenantID, tenantAdminEmail, tenantAdminPassword)
	require.NoError(t, err)

	// Get verification token and verify the account
	verificationToken, err := getVerificationTokenFromEmail(tenantAdminEmail)
	require.NoError(t, err)

	err = bulwarkGuard.Account.Verify(ctx, newTenantID, tenantAdminEmail, verificationToken)
	require.NoError(t, err)

	// Assign tenant_admin role via direct database access
	err = integration.SetupTestUserAsTenantAdmin(newTenantID, tenantAdminEmail)
	require.NoError(t, err)

	// Authenticate as the tenant admin user
	tenantAdminToken, err := integration.AuthenticateAsUser(t, newTenantID, tenantAdminEmail)
	require.NoError(t, err)

	// Now the tenant admin should be able to access their tenant's endpoints
	tenantBaseURL := fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), newTenantID)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, tenantBaseURL+"/accounts", tenantAdminToken, nil)
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

	resp, err := adminRequest("POST", "/tenants", adminTC, tenant1Payload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	tenant1ID := getTenantIDByName(adminTC, tenant1Name)
	require.NotEmpty(t, tenant1ID, "first tenant not found")

	// Create second tenant
	tenant2Name := fmt.Sprintf("Tenant2_%d", time.Now().UnixNano())
	tenant2Payload := map[string]string{
		"Name":        tenant2Name,
		"Description": "Second test tenant",
		"Domain":      "tenant2.example.com",
	}

	resp, err = adminRequest("POST", "/tenants", adminTC, tenant2Payload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	tenant2ID := getTenantIDByName(adminTC, tenant2Name)
	require.NotEmpty(t, tenant2ID, "second tenant not found")

	// Create and setup admin in tenant 1
	tenant1AdminEmail := fmt.Sprintf("admin1_%d@example.com", time.Now().UnixNano())
	tenant1AdminPassword := "TestPassword123!"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpClient := &http.Client{}
	bulwarkGuard := bulwark.NewGuard(integration.GetBulwarkAuthURL(), httpClient)

	err = bulwarkGuard.Account.Create(ctx, tenant1ID, tenant1AdminEmail, tenant1AdminPassword)
	require.NoError(t, err)

	verificationToken, err := getVerificationTokenFromEmail(tenant1AdminEmail)
	require.NoError(t, err)

	err = bulwarkGuard.Account.Verify(ctx, tenant1ID, tenant1AdminEmail, verificationToken)
	require.NoError(t, err)

	// Assign tenant_admin role
	err = integration.SetupTestUserAsTenantAdmin(tenant1ID, tenant1AdminEmail)
	require.NoError(t, err)

	// Authenticate as tenant 1 admin
	tenant1AdminToken, err := integration.AuthenticateAsUser(t, tenant1ID, tenant1AdminEmail)
	require.NoError(t, err)

	// Tenant 1 admin tries to access tenant 2 - should fail
	tenant2BaseURL := fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), tenant2ID)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, tenant2BaseURL+"/accounts", tenant1AdminToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 400 Bad Request (their JWT is for tenant1, not tenant2, so JWT validation fails)
	// or 403 Forbidden if the middleware catches it
	assert.True(t, resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest,
		"Expected 403 or 400, got %d", resp.StatusCode)
}

// TestSystemAdminCanAccessAnyTenant verifies that system admins can access admin endpoints for any tenant
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

	resp, err := adminRequest("POST", "/tenants", adminTC, tenantPayload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get the newly created tenant ID
	newTenantID := getTenantIDByName(adminTC, tenantName)
	require.NotEmpty(t, newTenantID, "newly created tenant not found")

	// System admin should be able to manage the new tenant via admin endpoints
	// Verify the tenant was created successfully by retrieving it
	adminURL := fmt.Sprintf("%s/api/v1/admin/tenants/%s", integration.GetBaseURL(), newTenantID)
	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, adminURL, adminTC.AccessToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should get 200 OK
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestTenantAdminRoleCreatedAutomatically verifies that tenant_admin role is created for new tenants
func TestTenantAdminRoleCreatedAutomatically(t *testing.T) {
	// Create a new tenant for this test (use SetupSystemAdminContext then call adminRequest)
	adminTC := integration.SetupSystemAdminContext(t)

	tenantName := fmt.Sprintf("RoleTest_%d", time.Now().UnixNano())
	tenantPayload := map[string]string{
		"Name":        tenantName,
		"Description": "Test tenant for role creation",
		"Domain":      "roletest.example.com",
	}

	resp, err := adminRequest("POST", "/tenants", adminTC, tenantPayload)
	require.NoError(t, err)
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("tenant creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Get the newly created tenant ID
	newTenantID := getTenantIDByName(adminTC, tenantName)
	require.NotEmpty(t, newTenantID, "newly created tenant not found")

	// Create a tenant admin in the new tenant
	tenantAdminEmail := fmt.Sprintf("admin_%d@example.com", time.Now().UnixNano())
	tenantAdminPassword := "TestPassword123!"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	httpClient := &http.Client{}
	bulwarkGuard := bulwark.NewGuard(integration.GetBulwarkAuthURL(), httpClient)

	err = bulwarkGuard.Account.Create(ctx, newTenantID, tenantAdminEmail, tenantAdminPassword)
	require.NoError(t, err)

	verificationToken, err := getVerificationTokenFromEmail(tenantAdminEmail)
	require.NoError(t, err)

	err = bulwarkGuard.Account.Verify(ctx, newTenantID, tenantAdminEmail, verificationToken)
	require.NoError(t, err)

	// Assign tenant_admin role
	err = integration.SetupTestUserAsTenantAdmin(newTenantID, tenantAdminEmail)
	require.NoError(t, err)

	// Authenticate as the tenant admin
	tenantAdminToken, err := integration.AuthenticateAsUser(t, newTenantID, tenantAdminEmail)
	require.NoError(t, err)

	// Tenant admin can now access roles and verify tenant_admin role exists
	tenantBaseURL := fmt.Sprintf("%s/api/v1/tenant/%s", integration.GetBaseURL(), newTenantID)
	rolesURL := tenantBaseURL + "/rbac/roles"

	resp, err = integration.MakeAuthenticatedRequest(http.MethodGet, rolesURL, tenantAdminToken, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list roles")

	var rolesResp []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&rolesResp)
	require.NoError(t, err)

	// Find the tenant_admin role
	foundTenantAdminRole := false
	for _, role := range rolesResp {
		if name, ok := role["name"].(string); ok && name == "tenant_admin" {
			foundTenantAdminRole = true
			break
		}
	}

	// Debug: if role not found, print what roles were returned
	if !foundTenantAdminRole {
		t.Logf("Admin role not found, available roles: %v (count: %d)", rolesResp, len(rolesResp))
		t.FailNow()
	}
}

// getTenantIDByName finds a tenant by name by listing all tenants
func getTenantIDByName(tc *integration.TestContext, tenantName string) string {
	resp, err := adminRequest("GET", "/tenants", tc, nil)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var tenantsList []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tenantsList); err != nil {
		return ""
	}

	for _, tenant := range tenantsList {
		if name, ok := tenant["name"].(string); ok && name == tenantName {
			if id, ok := tenant["id"].(string); ok {
				return id
			}
		}
	}
	return ""
}

// getVerificationTokenFromEmail extracts the verification token from mailhog for the given email
func getVerificationTokenFromEmail(email string) (string, error) {
	// Wait a bit for the email to arrive
	time.Sleep(500 * time.Millisecond)

	// Get messages from mailhog
	resp, err := http.Get("http://localhost:8025/api/v2/messages")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var mailhogResponse struct {
		Items []struct {
			Content struct {
				Body string `json:"Body"`
			} `json:"Content"`
			Raw struct {
				To []string `json:"To"`
			} `json:"Raw"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mailhogResponse); err != nil {
		return "", err
	}

	// Find the email for our test user and extract the verification token
	for _, item := range mailhogResponse.Items {
		for _, to := range item.Raw.To {
			if to == email {
				// Extract token from email body - look for verification URL pattern
				// The token is typically in a URL like: /verify?token=<token>
				body := item.Content.Body
				return extractTokenFromEmailBody(body)
			}
		}
	}

	return "", fmt.Errorf("verification email not found for %s", email)
}

// extractTokenFromEmailBody extracts the verification token from email body
func extractTokenFromEmailBody(body string) (string, error) {
	// Look for vt= (verification token) in the body
	// The email format is: ...&vt=<token>" or ...?vt=<token>...
	tokenStart := -1
	for i := 0; i < len(body)-3; i++ {
		if body[i:i+3] == "vt=" {
			tokenStart = i + 3
			break
		}
	}

	if tokenStart == -1 {
		return "", fmt.Errorf("verification token (vt=) not found in email body")
	}

	// Extract until whitespace, quote, ampersand, or end of string
	tokenEnd := tokenStart
	for tokenEnd < len(body) && body[tokenEnd] != ' ' && body[tokenEnd] != '\n' && body[tokenEnd] != '\r' && body[tokenEnd] != '"' && body[tokenEnd] != '&' && body[tokenEnd] != '<' {
		tokenEnd++
	}

	if tokenEnd == tokenStart {
		return "", fmt.Errorf("empty verification token in email body")
	}

	return body[tokenStart:tokenEnd], nil
}
