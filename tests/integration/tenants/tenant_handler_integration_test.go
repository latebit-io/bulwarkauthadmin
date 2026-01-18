//go:build integration

package tenants

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/tenants"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	integration.TestMain(m)
}

// adminRequest makes an authenticated request to the admin API
func adminRequest(method, path string, tc *integration.TestContext, payload interface{}) (*http.Response, error) {
	adminBaseURL := fmt.Sprintf("%s/api/v1/admin", integration.GetBaseURL())
	url := adminBaseURL + path

	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	return integration.MakeAuthenticatedRequest(method, url, tc.AccessToken, body)
}

func TestTenantHandler_AddTenant(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	payload := tenants.NewTenantRequest{
		Name:        fmt.Sprintf("Test Tenant %d", time.Now().UnixNano()),
		Description: "A test tenant created via integration test",
		Domain:      "test-tenant.example.com",
	}

	// Add tenant
	resp, err := adminRequest("POST", "/tenants", tc, payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Try adding same tenant name - should fail with conflict
	resp, err = adminRequest("POST", "/tenants", tc, payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestTenantHandler_AddTenant_MissingFields(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	tests := []struct {
		name           string
		payload        tenants.NewTenantRequest
		expectedStatus int
	}{
		{
			name: "Missing Name",
			payload: tenants.NewTenantRequest{
				Name:        "",
				Description: "A test tenant",
				Domain:      "test.example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Description",
			payload: tenants.NewTenantRequest{
				Name:        "Test Tenant",
				Description: "",
				Domain:      "test.example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Valid Request with Empty Domain",
			payload: tenants.NewTenantRequest{
				Name:        fmt.Sprintf("Valid Tenant %d", time.Now().UnixNano()),
				Description: "A valid tenant with empty domain",
				Domain:      "",
			},
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := adminRequest("POST", "/tenants", tc, tt.payload)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestTenantHandler_ListTenants(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	// Create a tenant first to ensure we have data
	payload := tenants.NewTenantRequest{
		Name:        fmt.Sprintf("List Test Tenant %d", time.Now().UnixNano()),
		Description: "For list testing",
		Domain:      "list-test.example.com",
	}
	resp, err := adminRequest("POST", "/tenants", tc, payload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// List tenants
	resp, err = adminRequest("GET", "/tenants", tc, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var tenantsList []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenantsList)
	require.NoError(t, err)

	// Should have at least one tenant
	assert.Greater(t, len(tenantsList), 0)

	// Verify our tenant is in the list
	found := false
	for _, tenant := range tenantsList {
		if tenant["name"] == payload.Name {
			found = true
			assert.Equal(t, payload.Description, tenant["description"])
			assert.Equal(t, payload.Domain, tenant["domain"])
			break
		}
	}
	assert.True(t, found, "Created tenant not found in list")
}

func TestTenantHandler_GetTenant(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	// Create a tenant
	payload := tenants.NewTenantRequest{
		Name:        fmt.Sprintf("Get Test Tenant %d", time.Now().UnixNano()),
		Description: "For get testing",
		Domain:      "get-test.example.com",
	}

	resp, err := adminRequest("POST", "/tenants", tc, payload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// List tenants to find our tenant ID
	resp, err = adminRequest("GET", "/tenants", tc, nil)
	require.NoError(t, err)

	var tenantsList []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenantsList)
	require.NoError(t, err)
	resp.Body.Close()

	// Find our tenant by name
	var tenantID string
	for _, ten := range tenantsList {
		if ten["name"] == payload.Name {
			tenantID = ten["id"].(string)
			break
		}
	}
	require.NotEmpty(t, tenantID, "Tenant not found in list")

	// Get the specific tenant
	resp, err = adminRequest("GET", fmt.Sprintf("/tenants/%s", tenantID), tc, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var retrievedTenant map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&retrievedTenant)
	require.NoError(t, err)

	assert.Equal(t, tenantID, retrievedTenant["id"])
	assert.Equal(t, payload.Name, retrievedTenant["name"])
	assert.Equal(t, payload.Description, retrievedTenant["description"])
	assert.Equal(t, payload.Domain, retrievedTenant["domain"])
}

func TestTenantHandler_GetTenant_NotFound(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	// Try to get a nonexistent tenant
	resp, err := adminRequest("GET", "/tenants/nonexistent-tenant-id", tc, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestTenantHandler_UpdateTenant(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	// Create a tenant
	payload := tenants.NewTenantRequest{
		Name:        fmt.Sprintf("Update Test Tenant %d", time.Now().UnixNano()),
		Description: "Original description",
		Domain:      "update-test.example.com",
	}

	resp, err := adminRequest("POST", "/tenants", tc, payload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get the tenant ID
	resp, err = adminRequest("GET", "/tenants", tc, nil)
	require.NoError(t, err)

	var tenantsList []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenantsList)
	require.NoError(t, err)
	resp.Body.Close()

	var tenantID string
	for _, ten := range tenantsList {
		if ten["name"] == payload.Name {
			tenantID = ten["id"].(string)
			break
		}
	}
	require.NotEmpty(t, tenantID)

	// Update the tenant
	updatePayload := tenants.NewTenantRequest{
		Name:        fmt.Sprintf("Updated Tenant %d", time.Now().UnixNano()),
		Description: "Updated description",
		Domain:      "updated-domain.example.com",
	}

	resp, err = adminRequest("PUT", fmt.Sprintf("/tenants/%s", tenantID), tc, updatePayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify the update
	resp, err = adminRequest("GET", fmt.Sprintf("/tenants/%s", tenantID), tc, nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	var retrievedTenant map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&retrievedTenant)
	require.NoError(t, err)

	assert.Equal(t, updatePayload.Name, retrievedTenant["name"])
	assert.Equal(t, updatePayload.Description, retrievedTenant["description"])
	assert.Equal(t, updatePayload.Domain, retrievedTenant["domain"])
}

func TestTenantHandler_UpdateTenant_MissingFields(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	// Create a tenant first
	payload := tenants.NewTenantRequest{
		Name:        fmt.Sprintf("Update Missing Fields Test %d", time.Now().UnixNano()),
		Description: "Original description",
		Domain:      "update-missing.example.com",
	}

	resp, err := adminRequest("POST", "/tenants", tc, payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get the tenant ID
	resp, err = adminRequest("GET", "/tenants", tc, nil)
	require.NoError(t, err)

	var tenantsList []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenantsList)
	require.NoError(t, err)
	resp.Body.Close()

	var tenantID string
	for _, ten := range tenantsList {
		if ten["name"] == payload.Name {
			tenantID = ten["id"].(string)
			break
		}
	}
	require.NotEmpty(t, tenantID)

	// Try to update with missing fields
	updatePayload := tenants.NewTenantRequest{
		Name:        "",
		Description: "Updated description",
		Domain:      "updated-domain.example.com",
	}

	resp, err = adminRequest("PUT", fmt.Sprintf("/tenants/%s", tenantID), tc, updatePayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTenantHandler_DeleteTenant(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	// Create a tenant
	payload := tenants.NewTenantRequest{
		Name:        fmt.Sprintf("Delete Test Tenant %d", time.Now().UnixNano()),
		Description: "For deletion testing",
		Domain:      "delete-test.example.com",
	}

	resp, err := adminRequest("POST", "/tenants", tc, payload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get the tenant ID
	resp, err = adminRequest("GET", "/tenants", tc, nil)
	require.NoError(t, err)

	var tenantsList []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tenantsList)
	require.NoError(t, err)
	resp.Body.Close()

	var tenantID string
	for _, ten := range tenantsList {
		if ten["name"] == payload.Name {
			tenantID = ten["id"].(string)
			break
		}
	}
	require.NotEmpty(t, tenantID)

	// Delete the tenant
	resp, err = adminRequest("DELETE", fmt.Sprintf("/tenants/%s", tenantID), tc, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify the tenant is deleted
	resp, err = adminRequest("GET", fmt.Sprintf("/tenants/%s", tenantID), tc, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestTenantHandler_DeleteTenant_NotFound(t *testing.T) {
	tc := integration.SetupSystemAdminContext(t)

	// Try to delete a nonexistent tenant
	resp, err := adminRequest("DELETE", "/tenants/nonexistent-tenant-id", tc, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
