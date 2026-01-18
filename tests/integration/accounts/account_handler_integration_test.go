//go:build integration

package accounts

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

func TestMain(m *testing.M) {
	integration.TestMain(m)
}

func TestAccountHandler_RegisterAccount(t *testing.T) {
	tc := integration.NewTestContext(t)

	email := fmt.Sprintf("user%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	// Register account
	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Try registering same email - should fail with conflict
	resp, err = tc.Post("/accounts", payload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestAccountHandler_ListAccounts(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account first to ensure we have data
	email := fmt.Sprintf("listtest%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// List accounts
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var accs []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&accs)
	require.NoError(t, err)

	// Should have at least one account
	assert.Greater(t, len(accs), 0)
}

func TestAccountHandler_GetAccount(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("gettest%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}

	resp, err := tc.Post("/accounts", payload)
	require.NoError(t, err)
	resp.Body.Close()

	// Get all accounts and find ours
	resp, err = tc.Get("/accounts")
	require.NoError(t, err)

	var accs []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&accs)
	require.NoError(t, err)
	resp.Body.Close()

	// Find our account by email
	var accountID string
	for _, acc := range accs {
		if acc["email"] == email {
			accountID = acc["id"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID, "Account not found in list")

	// Get the specific account
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var account map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&account)
	require.NoError(t, err)

	assert.Equal(t, email, account["email"])
}

func TestAccountHandler_ChangeEmail(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("changeemail%d@example.com", time.Now().UnixNano())
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

	// Change email
	newEmail := fmt.Sprintf("newemail%d@example.com", time.Now().UnixNano())
	changePayload := accounts.ChangeEmailRequest{
		AccountID: accountID,
		Email:     newEmail,
	}

	resp, err = tc.Put("/accounts/email", changePayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify email changed by getting the account
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	assert.Equal(t, newEmail, account["email"])
}

func TestAccountHandler_DeactivateAccount(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("deactivate%d@example.com", time.Now().UnixNano())
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

	// Deactivate account
	deactivatePayload := accounts.DeactivateAccountRequest{AccountID: accountID}

	resp, err = tc.Put("/accounts/deactivate", deactivatePayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify isDeleted flag
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	assert.Equal(t, true, account["isDeleted"])
}

func TestAccountHandler_DisableAccount(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("disable%d@example.com", time.Now().UnixNano())
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

	// Disable account
	disablePayload := accounts.DisableAccountRequest{AccountID: accountID}

	resp, err = tc.Put("/accounts/disable", disablePayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestAccountHandler_EnableAccount(t *testing.T) {
	tc := integration.NewTestContext(t)

	// Create an account
	email := fmt.Sprintf("enable%d@example.com", time.Now().UnixNano())
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

	// Disable first, then enable
	disablePayload := accounts.DisableAccountRequest{AccountID: accountID}
	resp, err = tc.Put("/accounts/disable", disablePayload)
	require.NoError(t, err)
	resp.Body.Close()

	// Enable account
	enablePayload := accounts.EnableAccountRequest{AccountID: accountID}

	resp, err = tc.Put("/accounts/enable", enablePayload)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify account is enabled
	resp, err = tc.Get("/accounts/" + accountID)
	require.NoError(t, err)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	assert.Equal(t, true, account["isEnabled"])
}
