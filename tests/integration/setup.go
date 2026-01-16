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
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient    *mongo.Client
	mongoURI       = "mongodb://localhost:27017/?directConnection=true"
	bulwarkAuthURL = "http://localhost:8080"
	baseURL        = "http://localhost:8081"
	mailhogURL     = "http://localhost:8025"

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

	// Create a test user account via BulwarkAuth
	testEmail := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	testPassword := "TestPassword123!"
	testClientID := "integration-test-client"

	// Register the account with bulwarkauth
	err = createAccount(tenantID, testEmail, testPassword)
	if err != nil {
		return "", "", fmt.Errorf("failed to create account: %w", err)
	}

	// For testing, directly mark the account as verified in the database
	err = markAccountAsVerified(tenantID, testEmail)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify account in database: %w", err)
	}

	// Authenticate and get access token
	accessToken, err := authenticateWithPassword(tenantID, testEmail, testPassword, testClientID)
	if err != nil {
		return "", "", fmt.Errorf("failed to authenticate: %w", err)
	}

	return tenantID, accessToken, nil
}

// ensureDefaultTenantExists creates the "default" tenant in bulwarkauthadmin's database
// if it doesn't already exist. This is needed because bulwarkauth uses "default" as its
// tenant ID, and bulwarkauthadmin needs this tenant to exist for the middleware to validate.
func ensureDefaultTenantExists() error {
	if mongoClient == nil {
		return errors.New("mongodb client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbName := "bulwarkauthtest"
	if seed := os.Getenv("DB_NAME_SEED"); seed != "" {
		dbName = "bulwarkauth" + seed
	}

	collection := mongoClient.Database(dbName).Collection("tenants")

	// Check if default tenant already exists
	var existing struct{}
	err := collection.FindOne(ctx, map[string]string{"id": DefaultTenantID}).Decode(&existing)
	if err == nil {
		// Tenant already exists
		return nil
	}

	// Create the default tenant
	tenant := map[string]interface{}{
		"id":          DefaultTenantID,
		"name":        "Default",
		"description": "Default tenant for testing",
		"domain":      "",
		"created":     time.Now(),
		"modified":    time.Now(),
	}

	_, err = collection.InsertOne(ctx, tenant)
	if err != nil {
		// Ignore duplicate key error (race condition)
		if !isDuplicateKeyError(err) {
			return err
		}
	}

	return nil
}

// isDuplicateKeyError checks if the error is a MongoDB duplicate key error
func isDuplicateKeyError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "E11000"))
}

// createAccount creates an account via BulwarkAuth API
func createAccount(tenantID, email, password string) error {
	payload := map[string]string{
		"tenantId": tenantID,
		"email":    email,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/accounts", bulwarkAuthURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create account: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getVerificationTokenFromMailhog retrieves the verification token from the email sent to mailhog
func getVerificationTokenFromMailhog(email string) (string, error) {
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

// markAccountAsVerified directly updates the database to mark an account as verified
// This bypasses the email verification flow for testing purposes
func markAccountAsVerified(tenantID, email string) error {
	if mongoClient == nil {
		return errors.New("mongodb client not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbName := "bulwarkauthtest"
	if seed := os.Getenv("DB_NAME_SEED"); seed != "" {
		dbName = "bulwarkauth" + seed
	}

	collection := mongoClient.Database(dbName).Collection("accounts")

	// Update the account to set isVerified=true and isEnabled=true
	filter := map[string]interface{}{
		"tenantId": tenantID,
		"email":    email,
	}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"isVerified": true,
			"isEnabled":  true,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("account not found: %s", email)
	}

	return nil
}

// verifyAccount verifies an account via BulwarkAuth API
func verifyAccount(tenantID, email, token string) error {
	payload := map[string]string{
		"tenantId":          tenantID,
		"email":             email,
		"verificationToken": token,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/accounts/verify", bulwarkAuthURL)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to verify account: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return nil
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

	dbName := "bulwarkauthtest"
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

// SetupSystemAdminContext creates a test context authenticated as the system admin
// The system admin account must be created via ADMIN_ACCOUNT and ADMIN_ACCOUNT_PASSWORD env vars
func SetupSystemAdminContext(t *testing.T) *TestContext {
	WaitForService(t, 20)
	WaitForBulwarkAuth(t, 20)

	// Get system admin credentials from environment
	adminEmail := os.Getenv("ADMIN_ACCOUNT")
	adminPassword := os.Getenv("ADMIN_ACCOUNT_PASSWORD")

	if adminEmail == "" || adminPassword == "" {
		t.Fatal("ADMIN_ACCOUNT and ADMIN_ACCOUNT_PASSWORD environment variables must be set for system admin tests")
	}

	// Authenticate as system admin via bulwarkauth
	// System admin is created in system tenant, but authenticates via bulwarkauth's default tenant
	accessToken, err := authenticateWithPassword(DefaultTenantID, adminEmail, adminPassword, "integration-test-client")
	if err != nil {
		t.Fatalf("Failed to authenticate as system admin: %v", err)
	}

	// System admin accesses the system tenant via the API
	return &TestContext{
		TenantID:    SystemTenantID,
		AccessToken: accessToken,
		BaseURL:     fmt.Sprintf("%s/api/v1/tenant/%s", baseURL, SystemTenantID),
		T:           t,
	}
}
