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
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

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

// createTestApiKey creates an API key for the given account and returns the apiKeyID
func createTestApiKey(t *testing.T, tc *integration.TestContext, accountID, keyName string) string {
	payload := apikeyapi.NewApiKeyRequest{Name: keyName}

	resp, err := tc.Post("/accounts/"+accountID+"/apikeys", payload)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// List keys to get the ID
	resp, err = tc.Get("/accounts/" + accountID + "/apikeys")
	require.NoError(t, err)
	defer resp.Body.Close()

	var keys []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&keys)
	require.NoError(t, err)

	for _, k := range keys {
		if k["name"] == keyName {
			return k["id"].(string)
		}
	}

	t.Fatalf("API key %q not found after creation", keyName)
	return ""
}

func TestAccountApiKeyHandler_ListApiKeys(t *testing.T) {
	tc := integration.NewTestContext(t)
	accountID := createTestAccount(t, tc)

	// Create two keys
	name1 := fmt.Sprintf("list-key-1-%d", time.Now().UnixNano())
	name2 := fmt.Sprintf("list-key-2-%d", time.Now().UnixNano())
	createTestApiKey(t, tc, accountID, name1)
	createTestApiKey(t, tc, accountID, name2)

	resp, err := tc.Get("/accounts/" + accountID + "/apikeys")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var keys []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&keys)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(keys), 2)

	// Verify both names are present
	names := make(map[string]bool)
	for _, k := range keys {
		names[k["name"].(string)] = true
	}
	assert.True(t, names[name1], "expected key %q in list", name1)
	assert.True(t, names[name2], "expected key %q in list", name2)
}

func TestAccountApiKeyHandler_GetApiKey(t *testing.T) {
	tc := integration.NewTestContext(t)
	accountID := createTestAccount(t, tc)

	keyName := fmt.Sprintf("get-key-%d", time.Now().UnixNano())
	apiKeyID := createTestApiKey(t, tc, accountID, keyName)

	resp, err := tc.Get("/accounts/" + accountID + "/apikeys/" + apiKeyID)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var key map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&key)
	require.NoError(t, err)
	assert.Equal(t, keyName, key["name"])
	assert.Equal(t, apiKeyID, key["id"])
	assert.Equal(t, accountID, key["accountId"])
}

func TestAccountApiKeyHandler_SuspendApiKey(t *testing.T) {
	tc := integration.NewTestContext(t)
	accountID := createTestAccount(t, tc)

	keyName := fmt.Sprintf("suspend-key-%d", time.Now().UnixNano())
	apiKeyID := createTestApiKey(t, tc, accountID, keyName)

	// Suspend the key
	resp, err := tc.Put("/accounts/"+accountID+"/apikeys/"+apiKeyID+"/suspend", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify it's disabled
	resp, err = tc.Get("/accounts/" + accountID + "/apikeys/" + apiKeyID)
	require.NoError(t, err)
	defer resp.Body.Close()

	var key map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&key)
	require.NoError(t, err)
	assert.Equal(t, false, key["isEnabled"])
}

func TestAccountApiKeyHandler_EnableApiKey(t *testing.T) {
	tc := integration.NewTestContext(t)
	accountID := createTestAccount(t, tc)

	keyName := fmt.Sprintf("enable-key-%d", time.Now().UnixNano())
	apiKeyID := createTestApiKey(t, tc, accountID, keyName)

	// Suspend first
	resp, err := tc.Put("/accounts/"+accountID+"/apikeys/"+apiKeyID+"/suspend", nil)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Enable the key
	resp, err = tc.Put("/accounts/"+accountID+"/apikeys/"+apiKeyID+"/enable", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify it's enabled
	resp, err = tc.Get("/accounts/" + accountID + "/apikeys/" + apiKeyID)
	require.NoError(t, err)
	defer resp.Body.Close()

	var key map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&key)
	require.NoError(t, err)
	assert.Equal(t, true, key["isEnabled"])
}

func TestAccountApiKeyHandler_RevokeApiKey(t *testing.T) {
	tc := integration.NewTestContext(t)
	accountID := createTestAccount(t, tc)

	keyName := fmt.Sprintf("revoke-key-%d", time.Now().UnixNano())
	apiKeyID := createTestApiKey(t, tc, accountID, keyName)

	// Revoke the key
	resp, err := tc.Delete("/accounts/" + accountID + "/apikeys/" + apiKeyID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify it's gone - listing should not contain the revoked key
	resp, err = tc.Get("/accounts/" + accountID + "/apikeys")
	require.NoError(t, err)
	defer resp.Body.Close()

	var keys []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&keys)
	require.NoError(t, err)

	for _, k := range keys {
		assert.NotEqual(t, apiKeyID, k["id"], "revoked key should not appear in list")
	}
}
