package tenants

import (
	"context"
	"errors"
	"testing"

	"github.com/latebit-io/bulwarkauthadmin/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

	return client, client.Database("bulwarkauth_test")
}

func cleanupMongoServer(t *testing.T, client *mongo.Client) {
	err := client.Disconnect(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
}

func TestMongoDbTenantRepository_Create(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	tests := []struct {
		name        string
		tenant      Tenant
		expectedErr bool
	}{
		{
			name: "Valid Tenant",
			tenant: Tenant{
				Name:        "Test Tenant",
				Description: "A test tenant",
				Domain:      "test.example.com",
			},
			expectedErr: false,
		},
		{
			name: "Tenant with Empty Name",
			tenant: Tenant{
				Name:        "",
				Description: "A test tenant",
				Domain:      "test.example.com",
			},
			expectedErr: false, // Repository doesn't validate, that's handler responsibility
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenantID, err := repo.Create(context.TODO(), tt.tenant)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Empty(t, tenantID)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, tenantID)
			}
		})
	}
}

func TestMongoDbTenantRepository_CreateDuplicate(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	// Create first tenant
	tenant := Tenant{
		Name:        "Duplicate Test",
		Description: "First tenant",
		Domain:      "first.example.com",
	}

	tenantID, err := repo.Create(context.TODO(), tenant)
	assert.NoError(t, err)
	assert.NotEmpty(t, tenantID)

	// Try creating same name - should fail
	tenantID, err = repo.Create(context.TODO(), tenant)
	assert.Error(t, err)
	assert.IsType(t, TenantDuplicateError{}, err)
	assert.Empty(t, tenantID)
}

func TestMongoDbTenantRepository_Read(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	// Create a test tenant
	tenant := Tenant{
		Name:        "Read Test Tenant",
		Description: "For read testing",
		Domain:      "read-test.example.com",
	}
	tenantID, err := repo.Create(context.TODO(), tenant)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		tenantID    string
		expectedErr bool
		checkFields bool
	}{
		{
			name:        "Valid Tenant ID",
			tenantID:    tenantID,
			expectedErr: false,
			checkFields: true,
		},
		{
			name:        "Invalid Tenant ID",
			tenantID:    "nonexistent-id",
			expectedErr: true,
			checkFields: false,
		},
		{
			name:        "Empty Tenant ID",
			tenantID:    "",
			expectedErr: true,
			checkFields: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.Read(context.TODO(), tt.tenantID)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, result)
				var tenantNotFoundErr TenantNotFoundError
				if tt.tenantID != "" {
					assert.True(t, errors.As(err, &tenantNotFoundErr))
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkFields {
					assert.Equal(t, tenant.Name, result.Name)
					assert.Equal(t, tenant.Description, result.Description)
					assert.Equal(t, tenant.Domain, result.Domain)
				}
			}
		})
	}
}

func TestMongoDbTenantRepository_ReadAll(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	// Create multiple test tenants
	tenants := []Tenant{
		{
			Name:        "Tenant 1",
			Description: "First tenant",
			Domain:      "tenant1.example.com",
		},
		{
			Name:        "Tenant 2",
			Description: "Second tenant",
			Domain:      "tenant2.example.com",
		},
		{
			Name:        "Tenant 3",
			Description: "Third tenant",
			Domain:      "tenant3.example.com",
		},
	}

	for _, tenant := range tenants {
		_, err := repo.Create(context.TODO(), tenant)
		assert.NoError(t, err)
	}

	// Read all tenants
	result, err := repo.ReadAll(context.TODO())
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(result), 3)

	// Verify that our tenants are in the results
	names := make(map[string]bool)
	for _, t := range result {
		names[t.Name] = true
	}

	for _, tenant := range tenants {
		assert.True(t, names[tenant.Name], "Tenant %s not found in results", tenant.Name)
	}
}

func TestMongoDbTenantRepository_Update(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	// Create a test tenant
	tenant := Tenant{
		Name:        "Update Test",
		Description: "Original description",
		Domain:      "update-test.example.com",
	}
	tenantID, err := repo.Create(context.TODO(), tenant)
	assert.NoError(t, err)

	// Update the tenant
	updatedTenant := Tenant{
		ID:          tenantID,
		Name:        "Updated Name",
		Description: "Updated description",
		Domain:      "updated-domain.example.com",
	}

	err = repo.Update(context.TODO(), updatedTenant)
	assert.NoError(t, err)

	// Verify the update
	result, err := repo.Read(context.TODO(), tenantID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", result.Name)
	assert.Equal(t, "Updated description", result.Description)
	assert.Equal(t, "updated-domain.example.com", result.Domain)
}

func TestMongoDbTenantRepository_Delete(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	// Create a test tenant
	tenant := Tenant{
		Name:        "Delete Test",
		Description: "For deletion testing",
		Domain:      "delete-test.example.com",
	}
	tenantID, err := repo.Create(context.TODO(), tenant)
	assert.NoError(t, err)

	// Delete the tenant
	err = repo.Delete(context.TODO(), tenantID)
	assert.NoError(t, err)

	// Verify the tenant is deleted
	result, err := repo.Read(context.TODO(), tenantID)
	assert.Error(t, err)
	assert.Nil(t, result)
	var tenantNotFoundErr TenantNotFoundError
	assert.True(t, errors.As(err, &tenantNotFoundErr))
}

func TestMongoDbTenantRepository_DeleteNonexistent(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	// Try to delete a nonexistent tenant
	err := repo.Delete(context.TODO(), "nonexistent-id")
	assert.Error(t, err)
	var tenantNotFoundErr TenantNotFoundError
	assert.True(t, errors.As(err, &tenantNotFoundErr))
}

func TestMongoDbTenantRepository_CreateSystem(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDbTenantRepository(db)

	// Create system tenant
	err := repo.CreateSystem(context.TODO())
	assert.NoError(t, err)

	// Verify system tenant was created
	result, err := repo.Read(context.TODO(), "00000000-0000-0000-0000-000000000000")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "System", result.Name)
	assert.Equal(t, "System tenant", result.Description)

	// Try creating system tenant again - should be idempotent
	err = repo.CreateSystem(context.TODO())
	assert.NoError(t, err)

	// Verify system tenant is still there
	result, err = repo.Read(context.TODO(), "00000000-0000-0000-0000-000000000000")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}
