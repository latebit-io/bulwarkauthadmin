//go:build integration

package accounts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/accounts"
	apikeyapi "github.com/latebit-io/bulwarkauthadmin/api/accounts/apikey"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestAccount registers an account and returns its ID
func createTestAccount(t *testing.T, tc *integration.TestContext) string {
	email := fmt.Sprintf("apikey_test_%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get account ID from list
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

	var accs []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&accs)
	require.NoError(t, err)
	resp.Body.Close()

	for _, acc := range accs {
		if acc["email"] == email {
			return acc["id"].(string)
		}
	}

	t.Fatal("Account not found after creation")
	return ""
}

func TestAccountApiKeyHandler_CreateApiKey(t *testing.T) {
	tc := integration.NewTestContext(t)
	accountID := createTestAccount(t, tc)

	keyName := fmt.Sprintf("test-key-%d", time.Now().UnixNano())
	payload := apikeyapi.NewApiKeyRequest{Name: keyName}

	resp, err := tc.Post("/accounts/"+accountID+"/apikeys", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var apiKey string
	err = json.NewDecoder(resp.Body).Decode(&apiKey)
	require.NoError(t, err)

	// Verify the key has the expected prefix
	assert.Contains(t, apiKey, "api_:")
}

func TestAccountApiKeyHandler_CreateApiKey_DuplicateName(t *testing.T) {
	tc := integration.NewTestContext(t)
	accountID := createTestAccount(t, tc)

	keyName := fmt.Sprintf("duplicate-key-%d", time.Now().UnixNano())
	payload := apikeyapi.NewApiKeyRequest{Name: keyName}

	// Create first key
	resp, err := tc.Post("/accounts/"+accountID+"/apikeys", payload)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Create duplicate - should fail with conflict
	resp, err = tc.Post("/accounts/"+accountID+"/apikeys", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestAccountApiKeyHandler_CreateApiKey_MissingAccountID(t *testing.T) {
	tc := integration.NewTestContext(t)

	payload := apikeyapi.NewApiKeyRequest{Name: "some-key"}

	// POST without account ID - should get 404 or 405 since route won't match
	resp, err := tc.Post("/accounts//apikeys", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.NotEqual(t, http.StatusCreated, resp.StatusCode)
}
