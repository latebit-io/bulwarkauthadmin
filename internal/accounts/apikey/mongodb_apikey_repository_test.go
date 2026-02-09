package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
	"github.com/latebit-io/bulwarkauthadmin/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const testTenantID = "00000000-0000-0000-0000-000000000001"
const testAccountID = "00000000-0000-0000-0000-000000000010"

func setupMongoServer(t *testing.T) (*mongo.Client, *mongo.Database) {
	mongodb := utils.NewMongoTestUtil()
	mongoServer, err := mongodb.CreateServer()
	if err != nil {
		t.Fatal(err)
	}

	clientOptions := options.Client().ApplyURI(mongoServer.URI())
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		t.Fatal(err)
	}

	return client, client.Database("bulwark")
}

func cleanupMongoServer(t *testing.T, client *mongo.Client) {
	err := client.Disconnect(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
}

func newTestApiKey(tenantID, accountID, name string) *ApiKey {
	expires := time.Now().Add(24 * time.Hour)
	return &ApiKey{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		AccountID: accountID,
		Name:      name,
		KeyHash:   "hashed_key_value",
		KeyPrefix: "api_",
		IsEnabled: true,
		Expires:   &expires,
		Created:   time.Now(),
		Modified:  time.Now(),
	}
}

func TestMongoDBApiKeyRepository_Create(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBApiKeyRepository(db)

	t.Run("Valid ApiKey", func(t *testing.T) {
		key := newTestApiKey(testTenantID, testAccountID, "my-key")
		err := repo.Create(context.TODO(), testTenantID, testAccountID, key)
		assert.NoError(t, err)
	})
}

func TestMongoDBApiKeyRepository_Read(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBApiKeyRepository(db)

	t.Run("Valid Read", func(t *testing.T) {
		key := newTestApiKey(testTenantID, testAccountID, "read-key")
		err := repo.Create(context.TODO(), testTenantID, testAccountID, key)
		assert.NoError(t, err)

		found, err := repo.Read(context.TODO(), key.ID, testTenantID, testAccountID)
		assert.NoError(t, err)
		assert.Equal(t, key.ID, found.ID)
		assert.Equal(t, key.Name, found.Name)
		assert.Equal(t, testTenantID, found.TenantID)
		assert.Equal(t, testAccountID, found.AccountID)
	})

	t.Run("Not Found", func(t *testing.T) {
		_, err := repo.Read(context.TODO(), "nonexistent-id", testTenantID, testAccountID)
		assert.Error(t, err)
	})
}

func TestMongoDBApiKeyRepository_Update(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBApiKeyRepository(db)

	t.Run("Update IsEnabled", func(t *testing.T) {
		key := newTestApiKey(testTenantID, testAccountID, "update-key")
		err := repo.Create(context.TODO(), testTenantID, testAccountID, key)
		assert.NoError(t, err)

		key.IsEnabled = false
		key.Modified = time.Now()
		err = repo.Update(context.TODO(), testTenantID, testAccountID, key)
		assert.NoError(t, err)

		updated, err := repo.Read(context.TODO(), key.ID, testTenantID, testAccountID)
		assert.NoError(t, err)
		assert.False(t, updated.IsEnabled)
	})
}

func TestMongoDBApiKeyRepository_Delete(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBApiKeyRepository(db)

	t.Run("Valid Delete", func(t *testing.T) {
		key := newTestApiKey(testTenantID, testAccountID, "delete-key")
		err := repo.Create(context.TODO(), testTenantID, testAccountID, key)
		assert.NoError(t, err)

		err = repo.Delete(context.TODO(), key.ID, testTenantID, testAccountID)
		assert.NoError(t, err)

		_, err = repo.Read(context.TODO(), key.ID, testTenantID, testAccountID)
		assert.Error(t, err)
	})

	t.Run("Delete Nonexistent Key", func(t *testing.T) {
		err := repo.Delete(context.TODO(), "nonexistent-id", testTenantID, testAccountID)
		assert.Error(t, err)
		assert.IsType(t, accounts.ApiKeyNotFoundError{}, err)
	})
}

func TestMongoDBApiKeyRepository_TenantIsolation(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBApiKeyRepository(db)

	tenant1 := "00000000-0000-0000-0000-000000000001"
	tenant2 := "00000000-0000-0000-0000-000000000002"

	key1 := newTestApiKey(tenant1, testAccountID, "shared-name")
	key2 := newTestApiKey(tenant2, testAccountID, "shared-name")

	err := repo.Create(context.TODO(), tenant1, testAccountID, key1)
	assert.NoError(t, err)

	err = repo.Create(context.TODO(), tenant2, testAccountID, key2)
	assert.NoError(t, err)

	// Read from tenant1 should only find key1
	found, err := repo.Read(context.TODO(), key1.ID, tenant1, testAccountID)
	assert.NoError(t, err)
	assert.Equal(t, tenant1, found.TenantID)

	// Read key1 from tenant2 should fail
	_, err = repo.Read(context.TODO(), key1.ID, tenant2, testAccountID)
	assert.Error(t, err)

	// Delete from tenant1 should not affect tenant2
	err = repo.Delete(context.TODO(), key1.ID, tenant1, testAccountID)
	assert.NoError(t, err)

	found, err = repo.Read(context.TODO(), key2.ID, tenant2, testAccountID)
	assert.NoError(t, err)
	assert.Equal(t, tenant2, found.TenantID)
}

func TestMongoDBApiKeyRepository_AccountIsolation(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBApiKeyRepository(db)

	account1 := "00000000-0000-0000-0000-000000000010"
	account2 := "00000000-0000-0000-0000-000000000020"

	key1 := newTestApiKey(testTenantID, account1, "my-key")
	key2 := newTestApiKey(testTenantID, account2, "my-key")

	err := repo.Create(context.TODO(), testTenantID, account1, key1)
	assert.NoError(t, err)

	err = repo.Create(context.TODO(), testTenantID, account2, key2)
	assert.NoError(t, err)

	// Read key1 with account2 should fail
	_, err = repo.Read(context.TODO(), key1.ID, testTenantID, account2)
	assert.Error(t, err)

	// Each account can read their own key
	found, err := repo.Read(context.TODO(), key1.ID, testTenantID, account1)
	assert.NoError(t, err)
	assert.Equal(t, account1, found.AccountID)
}
