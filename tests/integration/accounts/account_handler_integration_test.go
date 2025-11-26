//go:build integration
// +build integration

package accounts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/latebit-io/bulwarkauthadmin/api/accounts"
	"github.com/latebit-io/bulwarkauthadmin/tests/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Wait for service to be available
	integration.TestMain(m)
}

func TestAccountHandler_RegisterAccount(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()
	email := fmt.Sprintf("user%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	// Register account
	resp, err := http.Post(
		baseURL+"/api/accounts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// Try registering same email - should fail with conflict
	resp, err = http.Post(
		baseURL+"/api/accounts",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()
}

func TestAccountHandler_ListAccounts(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// List accounts
	resp, err := http.Get(baseURL + "/api/accounts")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var accs []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&accs)
	resp.Body.Close()

	// Should have at least some accounts (from other tests)
	assert.Greater(t, len(accs), 0)
}

func TestAccountHandler_GetAccount(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("gettest%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get all accounts and find ours
	resp, _ = http.Get(baseURL + "/api/accounts")
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var accs []map[string]interface{}
	json.Unmarshal(respBody, &accs)

	// Find our account by email
	var accountID string
	for _, acc := range accs {
		if acc["Email"] == email {
			accountID = acc["ID"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID, "Account not found in list")

	// Get the specific account
	resp, err := http.Get(baseURL + "/api/accounts/" + accountID)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	// Verify account details (note: single account endpoint returns lowercase fields)
	assert.Equal(t, email, account["email"])
}

func TestAccountHandler_ChangeEmail(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("changeemail%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/accounts")
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var accs []map[string]interface{}
	json.Unmarshal(respBody, &accs)

	var accountID string
	for _, acc := range accs {
		if acc["Email"] == email {
			accountID = acc["ID"].(string)
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
	changeBody, _ := json.Marshal(changePayload)

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/api/accounts/email", bytes.NewReader(changeBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify email changed by getting the account
	resp, _ = http.Get(baseURL + "/api/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	assert.Equal(t, newEmail, account["email"])
}

func TestAccountHandler_DeactivateAccount(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("deactivate%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/accounts")
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var accs []map[string]interface{}
	json.Unmarshal(respBody, &accs)

	var accountID string
	for _, acc := range accs {
		if acc["Email"] == email {
			accountID = acc["ID"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Deactivate account
	deactivatePayload := accounts.DeactivateAccountRequest{AccountID: accountID}
	deactivateBody, _ := json.Marshal(deactivatePayload)

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/api/accounts/deactivate", bytes.NewReader(deactivateBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify isDeleted flag
	resp, _ = http.Get(baseURL + "/api/accounts/" + accountID)
	var account map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&account)
	resp.Body.Close()

	assert.Equal(t, true, account["isDeleted"])
}

func TestAccountHandler_DisableAccount(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("disable%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/accounts")
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var accs []map[string]interface{}
	json.Unmarshal(respBody, &accs)

	var accountID string
	for _, acc := range accs {
		if acc["Email"] == email {
			accountID = acc["ID"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Disable account
	disablePayload := accounts.DisableAccountRequest{AccountID: accountID}
	disableBody, _ := json.Marshal(disablePayload)

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/api/accounts/disable", bytes.NewReader(disableBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}

func TestAccountHandler_EnableAccount(t *testing.T) {
	integration.WaitForService(t, 20)

	baseURL := integration.GetBaseURL()

	// Create an account
	email := fmt.Sprintf("enable%d@example.com", time.Now().UnixNano())
	payload := accounts.NewAccountRequest{Email: email}
	body, _ := json.Marshal(payload)

	resp, _ := http.Post(baseURL+"/api/accounts", "application/json", bytes.NewReader(body))
	resp.Body.Close()

	// Get account ID
	resp, _ = http.Get(baseURL + "/api/accounts")
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var accs []map[string]interface{}
	json.Unmarshal(respBody, &accs)

	var accountID string
	for _, acc := range accs {
		if acc["Email"] == email {
			accountID = acc["ID"].(string)
			break
		}
	}
	require.NotEmpty(t, accountID)

	// Enable account
	enablePayload := accounts.EnableAccountRequest{AccountID: accountID}
	enableBody, _ := json.Marshal(enablePayload)

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/api/accounts/enable", bytes.NewReader(enableBody))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}
