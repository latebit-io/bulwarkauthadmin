package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
	"github.com/latebit-io/bulwarkauthadmin/internal/utils"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const testRoleTenantID = "00000000-0000-0000-0000-000000000001"

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

func TestMongoDBRolesRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		role        Role
		expectedErr error
	}{
		{
			name:        "Valid Role",
			role:        NewRole(testRoleTenantID, "admin", "Administrator with full access"),
			expectedErr: nil,
		},
		{
			name: "Valid Role with Permissions",
			role: Role{
				TenantID:    testRoleTenantID,
				Name:        "moderator",
				Description: "Moderator role",
				Permissions: []string{"users:read", "posts:edit"},
			},
			expectedErr: nil,
		},
		{
			name:        "Duplicate Role",
			role:        NewRole(testRoleTenantID, "admin", "Administrator with full access"),
			expectedErr: RoleDuplicateError{Value: "admin"},
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBRolesRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(context.TODO(), testRoleTenantID, tt.role)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMongoDBRolesRepository_Read(t *testing.T) {
	tests := []struct {
		name             string
		roleName         string
		createRole       *Role
		expectedRoleName string
		expectedErr      bool
	}{
		{
			name:             "Valid Role",
			roleName:         "admin",
			createRole:       &Role{TenantID: testRoleTenantID, Name: "admin", Description: "Administrator"},
			expectedRoleName: "admin",
			expectedErr:      false,
		},
		{
			name:             "Role with Permissions",
			roleName:         "moderator",
			createRole:       &Role{TenantID: testRoleTenantID, Name: "moderator", Description: "Moderator", Permissions: []string{"users:read"}},
			expectedRoleName: "moderator",
			expectedErr:      false,
		},
		{
			name:        "Role Not Found",
			roleName:    "nonexistent",
			createRole:  nil,
			expectedErr: true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBRolesRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createRole != nil {
				err := repo.Create(context.TODO(), testRoleTenantID, *tt.createRole)
				assert.NoError(t, err)
			}

			role, err := repo.Read(context.TODO(), testRoleTenantID, tt.roleName)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr RoleNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRoleName, role.Name)
				if tt.createRole != nil && len(tt.createRole.Permissions) > 0 {
					assert.Equal(t, tt.createRole.Permissions, role.Permissions)
				}
			}
		})
	}
}

func TestMongoDBRolesRepository_Update(t *testing.T) {
	tests := []struct {
		name                string
		initialRole         Role
		updatedRole         Role
		expectedDescription string
		expectedPermissions []string
		expectedErr         bool
	}{
		{
			name:        "Update Description",
			initialRole: NewRole(testRoleTenantID, "admin", "Administrator"),
			updatedRole: Role{
				TenantID:    testRoleTenantID,
				Name:        "admin",
				Description: "Updated Administrator Description",
				Permissions: []string{},
			},
			expectedDescription: "Updated Administrator Description",
			expectedPermissions: []string{},
			expectedErr:         false,
		},
		{
			name:        "Add Permissions",
			initialRole: NewRole(testRoleTenantID, "editor", "Content Editor"),
			updatedRole: Role{
				TenantID:    testRoleTenantID,
				Name:        "editor",
				Description: "Content Editor",
				Permissions: []string{"posts:create", "posts:edit"},
			},
			expectedDescription: "Content Editor",
			expectedPermissions: []string{"posts:create", "posts:edit"},
			expectedErr:         false,
		},
		{
			name:        "Update Nonexistent Role",
			initialRole: Role{},
			updatedRole: Role{
				TenantID:    testRoleTenantID,
				Name:        "nonexistent",
				Description: "Should fail",
			},
			expectedErr: false, // MongoDB UpdateOne doesn't error if no match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, db := setupMongoServer(t)
			defer cleanupMongoServer(t, client)

			repo := NewMongoDBRolesRepository(db)

			if tt.initialRole.Name != "" {
				err := repo.Create(context.TODO(), testRoleTenantID, tt.initialRole)
				assert.NoError(t, err)
			}

			err := repo.Update(context.TODO(), testRoleTenantID, tt.updatedRole)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.initialRole.Name != "" {
					updatedRole, err := repo.Read(context.TODO(), testRoleTenantID, tt.updatedRole.Name)
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedDescription, updatedRole.Description)
					assert.Equal(t, tt.expectedPermissions, updatedRole.Permissions)
				}
			}
		})
	}
}

func TestMongoDBRolesRepository_ReadAll(t *testing.T) {
	tests := []struct {
		name          string
		roles         []Role
		expectedCount int
	}{
		{
			name: "Multiple Roles",
			roles: []Role{
				{TenantID: testRoleTenantID, Name: "admin", Description: "Administrator"},
				{TenantID: testRoleTenantID, Name: "moderator", Description: "Moderator"},
				{TenantID: testRoleTenantID, Name: "user", Description: "Standard User"},
			},
			expectedCount: 3,
		},
		{
			name:          "Empty Database",
			roles:         []Role{},
			expectedCount: 0,
		},
		{
			name: "Single Role",
			roles: []Role{
				{TenantID: testRoleTenantID, Name: "admin", Description: "Administrator"},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, db := setupMongoServer(t)
			defer cleanupMongoServer(t, client)

			repo := NewMongoDBRolesRepository(db)

			for _, role := range tt.roles {
				err := repo.Create(context.TODO(), testRoleTenantID, role)
				assert.NoError(t, err)
			}

			roles, err := repo.ReadAll(context.TODO(), testRoleTenantID, shared.NewPageOptions(0, 0, ""))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(roles))
		})
	}
}

func TestMongoDBRolesRepository_ReadAllWithPaging(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBRolesRepository(db)

	// Create multiple roles
	roles := []Role{
		{TenantID: testRoleTenantID, Name: "admin", Description: "Administrator"},
		{TenantID: testRoleTenantID, Name: "moderator", Description: "Moderator"},
		{TenantID: testRoleTenantID, Name: "editor", Description: "Editor"},
		{TenantID: testRoleTenantID, Name: "viewer", Description: "Viewer"},
		{TenantID: testRoleTenantID, Name: "guest", Description: "Guest"},
	}

	for _, role := range roles {
		err := repo.Create(context.TODO(), testRoleTenantID, role)
		assert.NoError(t, err)
	}

	// Test: Get first page (limit 2)
	page1, err := repo.ReadAll(context.TODO(), testRoleTenantID, shared.NewPageOptions(0, 2, ""))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page1))

	// Test: Get second page (offset 2, limit 2)
	page2, err := repo.ReadAll(context.TODO(), testRoleTenantID, shared.NewPageOptions(2, 2, ""))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page2))

	// Test: Get third page (offset 4, limit 2) - should return 1
	page3, err := repo.ReadAll(context.TODO(), testRoleTenantID, shared.NewPageOptions(4, 2, ""))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(page3))

	// Test: Get all (no limit)
	allRoles, err := repo.ReadAll(context.TODO(), testRoleTenantID, shared.NewPageOptions(0, 0, ""))
	assert.NoError(t, err)
	assert.Equal(t, 5, len(allRoles))
}

func TestMongoDBRolesRepository_Delete(t *testing.T) {
	tests := []struct {
		name        string
		roleName    string
		createRole  *Role
		expectedErr bool
	}{
		{
			name:        "Valid Delete",
			roleName:    "admin",
			createRole:  &Role{TenantID: testRoleTenantID, Name: "admin", Description: "Administrator"},
			expectedErr: false,
		},
		{
			name:        "Delete Nonexistent Role",
			roleName:    "nonexistent",
			createRole:  nil,
			expectedErr: true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBRolesRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createRole != nil {
				err := repo.Create(context.TODO(), testRoleTenantID, *tt.createRole)
				assert.NoError(t, err)
			}

			err := repo.Delete(context.TODO(), testRoleTenantID, tt.roleName)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr RoleNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)

				// Verify role is deleted
				_, err := repo.Read(context.TODO(), testRoleTenantID, tt.roleName)
				assert.Error(t, err)
				var notFoundErr RoleNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			}
		})
	}
}

func TestMongoDBRolesRepository_RolePermissionManagement(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBRolesRepository(db)

	// Create role
	role := NewRole(testRoleTenantID, "editor", "Content Editor")
	err := repo.Create(context.TODO(), testRoleTenantID, role)
	assert.NoError(t, err)

	// Add permission
	role.AddPermission("posts:create")
	role.AddPermission("posts:edit")
	err = repo.Update(context.TODO(), testRoleTenantID, role)
	assert.NoError(t, err)

	// Verify permissions added
	updatedRole, err := repo.Read(context.TODO(), testRoleTenantID, "editor")
	assert.NoError(t, err)
	assert.Len(t, updatedRole.Permissions, 2)
	assert.Contains(t, updatedRole.Permissions, "posts:create")
	assert.Contains(t, updatedRole.Permissions, "posts:edit")

	// Remove permission
	updatedRole.RemovePermission("posts:create")
	err = repo.Update(context.TODO(), testRoleTenantID, *updatedRole)
	assert.NoError(t, err)

	// Verify permission removed
	finalRole, err := repo.Read(context.TODO(), testRoleTenantID, "editor")
	assert.NoError(t, err)
	assert.Len(t, finalRole.Permissions, 1)
	assert.Contains(t, finalRole.Permissions, "posts:edit")
	assert.NotContains(t, finalRole.Permissions, "posts:create")
}

func TestMongoDBRolesRepository_TenantIsolation(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBRolesRepository(db)

	tenant1 := "00000000-0000-0000-0000-000000000001"
	tenant2 := "00000000-0000-0000-0000-000000000002"

	// Create same role name in two different tenants
	role1 := Role{TenantID: tenant1, Name: "admin", Description: "Admin for tenant 1"}
	err := repo.Create(context.TODO(), tenant1, role1)
	assert.NoError(t, err)

	role2 := Role{TenantID: tenant2, Name: "admin", Description: "Admin for tenant 2"}
	err = repo.Create(context.TODO(), tenant2, role2)
	assert.NoError(t, err)

	// Verify roles are isolated by tenant
	foundRole1, err := repo.Read(context.TODO(), tenant1, "admin")
	assert.NoError(t, err)
	assert.Equal(t, tenant1, foundRole1.TenantID)
	assert.Equal(t, "Admin for tenant 1", foundRole1.Description)

	foundRole2, err := repo.Read(context.TODO(), tenant2, "admin")
	assert.NoError(t, err)
	assert.Equal(t, tenant2, foundRole2.TenantID)
	assert.Equal(t, "Admin for tenant 2", foundRole2.Description)

	// Verify ReadAll returns only roles for the specified tenant
	roles1, err := repo.ReadAll(context.TODO(), tenant1, shared.NewPageOptions(0, 0, ""))
	assert.NoError(t, err)
	assert.Len(t, roles1, 1)
	assert.Equal(t, tenant1, roles1[0].TenantID)

	roles2, err := repo.ReadAll(context.TODO(), tenant2, shared.NewPageOptions(0, 0, ""))
	assert.NoError(t, err)
	assert.Len(t, roles2, 1)
	assert.Equal(t, tenant2, roles2[0].TenantID)

	// Delete from one tenant shouldn't affect another
	err = repo.Delete(context.TODO(), tenant1, "admin")
	assert.NoError(t, err)

	// Tenant1's role is gone
	_, err = repo.Read(context.TODO(), tenant1, "admin")
	assert.Error(t, err)

	// Tenant2's role still exists
	_, err = repo.Read(context.TODO(), tenant2, "admin")
	assert.NoError(t, err)
}
