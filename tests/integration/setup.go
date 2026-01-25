//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	bulwark "github.com/latebit-io/bulwark-auth-guard"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient    *mongo.Client
	mongoURI       = "mongodb://localhost:27017/?directConnection=true"
	bulwarkAuthURL = "http://localhost:8080"
	baseURL        = "http://localhost:8081"
	mailhogURL     = "http://localhost:8025"
	bulwarkGuard   *bulwark.Guard

	// Test tenant and auth state
	testTenantID    string
	testAccessToken string
	testSetupOnce   sync.Once
	testSetupErr    error

	// System tenant ID (UUID nil) - used by bulwarkauthadmin
	SystemTenantID = "00000000-0000-0000-0000-000000000000"

	// Default tenant ID - used by bulwarkauth for authentication
	DefaultTenantID = "default"
)

func init() {
	if url := os.Getenv("BULWARK_AUTH_URL"); url != "" {
		bulwarkAuthURL = url
	}
	if url := os.Getenv("BULWARK_ADMIN_URL"); url != "" {
		baseURL = url
	}
	if url := os.Getenv("MAILHOG_URL"); url != "" {
		mailhogURL = url
	}
	if uri := os.Getenv("DB_CONNECTION"); uri != "" {
		mongoURI = uri
	}

	// Initialize the bulwark guard
	httpClient := &http.Client{}
	bulwarkGuard = bulwark.NewGuard(bulwarkAuthURL, httpClient)
}

func TestMain(m *testing.M) {
	// Wait for MongoDB to be ready
	if err := waitForMongoDB(30); err != nil {
		log.Printf("MongoDB not available: %v. Skipping integration tests.", err)
		os.Exit(0)
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v. Skipping integration tests.", err)
		os.Exit(0)
	}

	// Verify connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Printf("MongoDB ping failed: %v. Skipping integration tests.", err)
		os.Exit(0)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := mongoClient.Disconnect(context.Background()); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	}

	os.Exit(code)
}

// waitForMongoDB waits for MongoDB to be available
func waitForMongoDB(maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
		cancel()

		if err == nil {
			if err := client.Ping(context.Background(), nil); err == nil {
				client.Disconnect(context.Background())
				return nil
			}
			client.Disconnect(context.Background())
		}

		time.Sleep(1 * time.Second)
	}

	return errors.New("MongoDB did not become available in time")
}

// WaitForService waits for the BulwarkAuthAdmin service to be available
func WaitForService(t *testing.T, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("BulwarkAuthAdmin service did not become available in time")
}

// WaitForBulwarkAuth waits for the BulwarkAuth service to be available
func WaitForBulwarkAuth(t *testing.T, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(bulwarkAuthURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("BulwarkAuth service did not become available in time")
}

// GetBaseURL returns the base URL of the BulwarkAuthAdmin test service
func GetBaseURL() string {
	return baseURL
}

// GetBulwarkAuthURL returns the base URL of the BulwarkAuth service
func GetBulwarkAuthURL() string {
	return bulwarkAuthURL
}

// GetSystemTenantID returns the system tenant ID (UUID nil)
func GetSystemTenantID() string {
	return SystemTenantID
}

// SetupTestTenant creates a test tenant and registers an authenticated user
// This is called once per test run and caches the results
func SetupTestTenant(t *testing.T) (tenantID, accessToken string) {
	testSetupOnce.Do(func() {
		testTenantID, testAccessToken, testSetupErr = setupTestTenantInternal(t)
	})

	if testSetupErr != nil {
		t.Fatalf("Failed to setup test tenant: %v", testSetupErr)
	}

	return testTenantID, testAccessToken
}

// setupTestTenantInternal does the actual work of setting up test infrastructure
func setupTestTenantInternal(t *testing.T) (string, string, error) {
	// Use the "default" tenant that bulwarkauth creates on startup
	// The JWT token will contain this tenant ID, and we must use the same
	// tenant ID when calling bulwarkauthadmin APIs for the middleware to validate
	tenantID := DefaultTenantID

	// Ensure the "default" tenant exists in bulwarkauthadmin's tenant table
	err := ensureDefaultTenantExists()
	if err != nil {
		return "", "", fmt.Errorf("failed to ensure default tenant exists: %w", err)
	}

	// Create a test user account via BulwarkAuth using the Guard client
	testEmail := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	testPassword := "TestPassword123!"
	testClientID := "integration-test-client"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Register the account with bulwarkauth using the Guard
	err = bulwarkGuard.Account.Create(ctx, tenantID, testEmail, testPassword)
	if err != nil {
		return "", "", fmt.Errorf("failed to create account: %w", err)
	}

	// Get verification token from mailhog and verify via BulwarkAuth API
	verificationToken, err := GetVerificationTokenFromEmail(testEmail)
	if err != nil {
		return "", "", fmt.Errorf("failed to get verification token from mailhog: %w", err)
	}

	err = bulwarkGuard.Account.Verify(ctx, tenantID, testEmail, verificationToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify account: %w", err)
	}

	// Assign tenant_admin role to the test user BEFORE authenticating
	// This way, when bulwarkauth issues a JWT, it will include the tenant_admin role from the shared database
	err = SetupTestUserAsTenantAdmin(tenantID, testEmail)
	if err != nil {
		return "", "", fmt.Errorf("failed to setup test user as tenant admin: %w", err)
	}

	// Now authenticate and get access token - the JWT will include the tenant_admin role
	accessToken, err := authenticateWithPassword(tenantID, testEmail, testPassword, testClientID)
	if err != nil {
		return "", "", fmt.Errorf("failed to authenticate: %w", err)
	}

	return tenantID, accessToken, nil
}

// ensureDefaultTenantExists creates the "default" tenant in bulwarkauthadmin via the admin API.
// This is needed because bulwarkauth uses "default" as its tenant ID, and bulwarkauthadmin
// needs this tenant to exist for the middleware to validate.
// Uses system admin authentication to call the tenant management API.
func ensureDefaultTenantExists() error {
	// Get system admin credentials from environment
	adminEmail := os.Getenv("ADMIN_ACCOUNT")
	adminPassword := os.Getenv("ADMIN_ACCOUNT_PASSWORD")

	if adminEmail == "" || adminPassword == "" {
		return errors.New("ADMIN_ACCOUNT and ADMIN_ACCOUNT_PASSWORD environment variables must be set")
	}

	// Authenticate as system admin to get access token
	var accessToken string
	var err error
	for i := 0; i < 10; i++ {
		accessToken, err = authenticateWithPassword(SystemTenantID, adminEmail, adminPassword, "integration-test-client")
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("failed to authenticate as system admin: %w", err)
	}

	// Check if default tenant already exists via API
	adminURL := fmt.Sprintf("%s/api/v1/admin/tenants/%s", baseURL, DefaultTenantID)
	resp, err := MakeAuthenticatedRequest(http.MethodGet, adminURL, accessToken, nil)
	if err != nil {
		return fmt.Errorf("failed to check tenant existence: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Tenant already exists
		return nil
	}

	// Create the default tenant via admin API
	createURL := fmt.Sprintf("%s/api/v1/admin/tenants", baseURL)
	tenant := map[string]string{
		"Name":        "Default",
		"Description": "Default tenant for testing",
		"Domain":      "",
	}
	body, _ := json.Marshal(tenant)

	resp, err = MakeAuthenticatedRequest(http.MethodPost, createURL, accessToken, body)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}
	defer resp.Body.Close()

	// 201 Created or 409 Conflict (already exists) are both acceptable
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create tenant: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetVerificationTokenFromEmail retrieves the verification token from the email sent to mailhog
func GetVerificationTokenFromEmail(email string) (string, error) {
	// Wait a bit for the email to arrive
	time.Sleep(500 * time.Millisecond)

	// Get messages from mailhog
	resp, err := http.Get(mailhogURL + "/api/v2/messages")
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
				return ExtractTokenFromEmailBody(body)
			}
		}
	}

	return "", fmt.Errorf("verification email not found for %s", email)
}

// authenticateWithPassword authenticates via BulwarkAuth and returns access token
func authenticateWithPassword(tenantID, email, password, clientID string) (string, error) {
	payload := map[string]string{
		"tenantId": tenantID,
		"email":    email,
		"password": password,
		"clientId": clientID,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/authenticate", bulwarkAuthURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to authenticate: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var authResponse struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return "", err
	}

	// Acknowledge the token
	ackPayload := map[string]string{
		"tenantId":     tenantID,
		"accessToken":  authResponse.AccessToken,
		"refreshToken": authResponse.RefreshToken,
	}
	ackBody, _ := json.Marshal(ackPayload)

	ackURL := fmt.Sprintf("%s/api/authenticate/ack", bulwarkAuthURL)
	ackResp, err := http.Post(ackURL, "application/json", bytes.NewReader(ackBody))
	if err != nil {
		return "", err
	}
	defer ackResp.Body.Close()

	return authResponse.AccessToken, nil
}

// ExtractTokenFromEmailBody extracts the verification token from email body
func ExtractTokenFromEmailBody(body string) (string, error) {
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
		return "", errors.New("verification token (vt=) not found in email body")
	}

	// Extract until whitespace, quote, ampersand, or end of string
	tokenEnd := tokenStart
	for tokenEnd < len(body) && body[tokenEnd] != ' ' && body[tokenEnd] != '\n' && body[tokenEnd] != '\r' && body[tokenEnd] != '"' && body[tokenEnd] != '&' && body[tokenEnd] != '<' {
		tokenEnd++
	}

	if tokenEnd == tokenStart {
		return "", errors.New("empty verification token in email body")
	}

	return body[tokenStart:tokenEnd], nil
}

// MakeAuthenticatedRequest makes an HTTP request with the Authorization header
func MakeAuthenticatedRequest(method, url, accessToken string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	return client.Do(req)
}

// GetTenantURL returns the tenant-scoped base URL
func GetTenantURL(tenantID string) string {
	return fmt.Sprintf("%s/api/v1/tenant/%s", baseURL, tenantID)
}

// CleanupDatabase clears all data from the test database
func CleanupDatabase(t *testing.T) {
	if mongoClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbName := "bulwarkauth"
	if seed := os.Getenv("DB_NAME_SEED"); seed != "" {
		dbName = "bulwarkauth" + seed
	}

	db := mongoClient.Database(dbName)

	// Drop all collections
	collections := []string{"accounts", "roles", "permissions", "tenants"}
	for _, coll := range collections {
		if err := db.Collection(coll).Drop(ctx); err != nil {
			// It's okay if the collection doesn't exist
			t.Logf("Note: Could not drop collection %s: %v", coll, err)
		}
	}
}

// TestContext holds common test state
type TestContext struct {
	TenantID    string
	AccessToken string
	BaseURL     string
	T           *testing.T
}

// NewTestContext creates a new test context with authentication
func NewTestContext(t *testing.T) *TestContext {
	WaitForService(t, 20)
	WaitForBulwarkAuth(t, 20)

	tenantID, accessToken := SetupTestTenant(t)

	return &TestContext{
		TenantID:    tenantID,
		AccessToken: accessToken,
		BaseURL:     GetTenantURL(tenantID),
		T:           t,
	}
}

// Post makes an authenticated POST request
func (tc *TestContext) Post(path string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return MakeAuthenticatedRequest(http.MethodPost, tc.BaseURL+path, tc.AccessToken, body)
}

// Get makes an authenticated GET request
func (tc *TestContext) Get(path string) (*http.Response, error) {
	return MakeAuthenticatedRequest(http.MethodGet, tc.BaseURL+path, tc.AccessToken, nil)
}

// Put makes an authenticated PUT request
func (tc *TestContext) Put(path string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return MakeAuthenticatedRequest(http.MethodPut, tc.BaseURL+path, tc.AccessToken, body)
}

// Patch makes an authenticated PATCH request
func (tc *TestContext) Patch(path string, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return MakeAuthenticatedRequest(http.MethodPatch, tc.BaseURL+path, tc.AccessToken, body)
}

// Delete makes an authenticated DELETE request
func (tc *TestContext) Delete(path string) (*http.Response, error) {
	return MakeAuthenticatedRequest(http.MethodDelete, tc.BaseURL+path, tc.AccessToken, nil)
}

// AuthenticateAsUser authenticates a user in a specific tenant and returns their access token
// This is used to test as different users in integration tests
func AuthenticateAsUser(t *testing.T, tenantID, email string) (string, error) {
	// For testing, we use a default password that's set when accounts are created
	// In a real scenario, you'd need to know or set the user's password
	// For integration tests, we'll create a password and use it
	password := "TestPassword123!"

	// Try to authenticate with the test password
	// If the user doesn't have this password set, they need to be created first
	accessToken, err := authenticateWithPassword(tenantID, email, password, "integration-test-client")
	if err != nil {
		return "", fmt.Errorf("failed to authenticate user %s in tenant %s: %w", email, tenantID, err)
	}

	return accessToken, nil
}

// SetupSystemAdminContext creates a test context authenticated as the system admin
// The system admin account must be created via ADMIN_ACCOUNT and ADMIN_ACCOUNT_PASSWORD env vars
func SetupSystemAdminContext(t *testing.T) *TestContext {
	WaitForService(t, 20)
	WaitForBulwarkAuth(t, 20)

	adminEmail := "admin@test.example.com"
	adminPassword := "TestAdminPassword123!"
	// Get system admin credentials from environment
	// adminEmail := os.Getenv("ADMIN_ACCOUNT")
	// adminPassword := os.Getenv("ADMIN_ACCOUNT_PASSWORD")

	if adminEmail == "" || adminPassword == "" {
		t.Fatal("ADMIN_ACCOUNT and ADMIN_ACCOUNT_PASSWORD environment variables must be set for system admin tests")
	}

	// The system admin account is created and auto-verified by bulwarkauthadmin on startup
	// via the AdminAccountsService.RegisterAccount() method

	// Authenticate as system admin via bulwarkauth
	// System admin is created in the system tenant (UUID nil)
	// Retry a few times in case the account needs a moment to be available
	var accessToken string
	var err error
	for i := 0; i < 5; i++ {
		accessToken, err = authenticateWithPassword(SystemTenantID, adminEmail, adminPassword, "integration-test-client")
		if err == nil {
			break
		}
		if i < 4 {
			t.Logf("Admin authentication attempt %d failed, retrying: %v", i+1, err)
			time.Sleep(500 * time.Millisecond)
		}
	}

	if err != nil {
		t.Fatalf("Failed to authenticate as system admin after retries: %v", err)
	}

	// System admin accesses the system tenant via the API
	return &TestContext{
		TenantID:    SystemTenantID,
		AccessToken: accessToken,
		BaseURL:     fmt.Sprintf("%s/api/v1/tenant/%s", baseURL, SystemTenantID),
		T:           t,
	}
}

// SetupTestUserAsTenantAdmin creates a test user account in the database and assigns them the tenant_admin role
// This is done via direct database access to bypass JWT validation issues when cross-tenant API calls are made
func SetupTestUserAsTenantAdmin(tenantID, email string) error {
	if mongoClient == nil {
		return errors.New("MongoDB client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbName := "bulwarkauth"
	if seed := os.Getenv("DB_NAME_SEED"); seed != "" {
		dbName = "bulwarkauth" + seed
	}

	db := mongoClient.Database(dbName)
	accountsCollection := db.Collection("accounts")

	now := time.Now()

	// First try to just add the tenant_admin role to any existing account with this email in this tenant
	filter := map[string]interface{}{
		"tenantId": tenantID,
		"email":    email,
	}

	update := map[string]interface{}{
		"$addToSet": map[string]interface{}{
			"roles": "tenant_admin",
		},
		"$set": map[string]interface{}{
			"modified": now,
		},
	}

	result, err := accountsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// If no document was matched, create a new one
	if result.MatchedCount == 0 {
		testUserID := uuid.New().String()
		account := map[string]interface{}{
			"_id":               testUserID,
			"tenantId":          tenantID,
			"email":             email,
			"isVerified":        true,
			"verificationToken": "",
			"isEnabled":         true,
			"isDeleted":         false,
			"socialProviders":   []interface{}{},
			"roles":             []string{"tenant_admin"},
			"permissions":       []interface{}{},
			"created":           now,
			"modified":          now,
		}

		_, err := accountsCollection.InsertOne(ctx, account)
		return err
	}

	return nil
}
