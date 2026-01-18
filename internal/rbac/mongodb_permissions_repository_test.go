package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
	"github.com/stretchr/testify/assert"
)

const testTenantID = "00000000-0000-0000-0000-000000000001"

func TestMongoDBPermissionsRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		permission  Permission
		expectedErr error
	}{
		{
			name:        "Valid Permission",
			permission:  NewPermission(testTenantID, "users", "read"),
			expectedErr: nil,
		},
		{
			name:        "Valid Permission with Action",
			permission:  NewPermission(testTenantID, "posts", "create"),
			expectedErr: nil,
		},
		{
			name: "Duplicate Permission",
			permission: Permission{
				TenantID: testTenantID,
				Key:      "users:read",
				Name:     "users",
				Action:   "read",
			},
			expectedErr: PermissionDuplicateError{Value: "users:read"},
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	// Create first permission for duplicate test
	firstPerm := Permission{
		TenantID: testTenantID,
		Key:      "users:read",
		Name:     "users",
		Action:   "read",
	}
	err := repo.Create(context.TODO(), testTenantID, firstPerm)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(context.TODO(), testTenantID, tt.permission)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMongoDBPermissionsRepository_Read(t *testing.T) {
	tests := []struct {
		name             string
		permissionKey    string
		createPermission *Permission
		expectedName     string
		expectedAction   string
		expectedErr      bool
	}{
		{
			name:          "Valid Permission",
			permissionKey: "users:read",
			createPermission: &Permission{
				TenantID: testTenantID,
				Key:      "users:read",
				Name:     "users",
				Action:   "read",
			},
			expectedName:   "users",
			expectedAction: "read",
			expectedErr:    false,
		},
		{
			name:          "Permission with Different Action",
			permissionKey: "posts:delete",
			createPermission: &Permission{
				TenantID: testTenantID,
				Key:      "posts:delete",
				Name:     "posts",
				Action:   "delete",
			},
			expectedName:   "posts",
			expectedAction: "delete",
			expectedErr:    false,
		},
		{
			name:             "Permission Not Found",
			permissionKey:    "nonexistent:action",
			createPermission: nil,
			expectedErr:      true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createPermission != nil {
				err := repo.Create(context.TODO(), testTenantID, *tt.createPermission)
				assert.NoError(t, err)
			}

			permission, err := repo.Read(context.TODO(), testTenantID, tt.permissionKey)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr PermissionNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedName, permission.Name)
				assert.Equal(t, tt.expectedAction, permission.Action)
				assert.Equal(t, tt.permissionKey, permission.Key)
			}
		})
	}
}

func TestMongoDBPermissionsRepository_ReadAll(t *testing.T) {
	tests := []struct {
		name          string
		permissions   []Permission
		expectedCount int
	}{
		{
			name: "Multiple Permissions",
			permissions: []Permission{
				{TenantID: testTenantID, Key: "users:read", Name: "users", Action: "read"},
				{TenantID: testTenantID, Key: "users:write", Name: "users", Action: "write"},
				{TenantID: testTenantID, Key: "posts:delete", Name: "posts", Action: "delete"},
			},
			expectedCount: 3,
		},
		{
			name:          "Empty Database",
			permissions:   []Permission{},
			expectedCount: 0,
		},
		{
			name: "Single Permission",
			permissions: []Permission{
				{TenantID: testTenantID, Key: "users:read", Name: "users", Action: "read"},
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, db := setupMongoServer(t)
			defer cleanupMongoServer(t, client)

			repo := NewMongoDBPermissionsRepository(db)

			for _, perm := range tt.permissions {
				err := repo.Create(context.TODO(), testTenantID, perm)
				assert.NoError(t, err)
			}

			permissions, err := repo.ReadAll(context.TODO(), testTenantID, shared.NewPageOptions(0, 0, ""))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedCount, len(permissions))
		})
	}
}

func TestMongoDBPermissionsRepository_Delete(t *testing.T) {
	tests := []struct {
		name             string
		permissionKey    string
		createPermission *Permission
		expectedErr      bool
	}{
		{
			name:          "Valid Delete",
			permissionKey: "users:read",
			createPermission: &Permission{
				TenantID: testTenantID,
				Key:      "users:read",
				Name:     "users",
				Action:   "read",
			},
			expectedErr: false,
		},
		{
			name:             "Delete Nonexistent Permission",
			permissionKey:    "nonexistent:action",
			createPermission: nil,
			expectedErr:      true,
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createPermission != nil {
				err := repo.Create(context.TODO(), testTenantID, *tt.createPermission)
				assert.NoError(t, err)
			}

			err := repo.Delete(context.TODO(), testTenantID, tt.permissionKey)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr PermissionNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)

				// Verify permission is deleted
				_, err := repo.Read(context.TODO(), testTenantID, tt.permissionKey)
				assert.Error(t, err)
				var notFoundErr PermissionNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			}
		})
	}
}

func TestMongoDBPermissionsRepository_ReadAllWithPaging(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	// Create multiple permissions
	permissions := []Permission{
		{TenantID: testTenantID, Key: "users:read", Name: "users", Action: "read"},
		{TenantID: testTenantID, Key: "users:write", Name: "users", Action: "write"},
		{TenantID: testTenantID, Key: "users:delete", Name: "users", Action: "delete"},
		{TenantID: testTenantID, Key: "posts:read", Name: "posts", Action: "read"},
		{TenantID: testTenantID, Key: "posts:write", Name: "posts", Action: "write"},
	}

	for _, perm := range permissions {
		err := repo.Create(context.TODO(), testTenantID, perm)
		assert.NoError(t, err)
	}

	// Test ReadAll returns all
	allPerms, err := repo.ReadAll(context.TODO(), testTenantID, shared.NewPageOptions(0, 0, ""))
	assert.NoError(t, err)
	assert.Equal(t, 5, len(allPerms))
}

func TestMongoDBPermissionsRepository_MultipleActionsOnSameResource(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	// Create multiple actions on same resource
	permissions := []Permission{
		{TenantID: testTenantID, Key: "users:read", Name: "users", Action: "read"},
		{TenantID: testTenantID, Key: "users:write", Name: "users", Action: "write"},
		{TenantID: testTenantID, Key: "users:delete", Name: "users", Action: "delete"},
	}

	for _, perm := range permissions {
		err := repo.Create(context.TODO(), testTenantID, perm)
		assert.NoError(t, err)
	}

	// Verify all permissions exist
	for _, perm := range permissions {
		foundPerm, err := repo.Read(context.TODO(), testTenantID, perm.Key)
		assert.NoError(t, err)
		assert.Equal(t, perm.Name, foundPerm.Name)
		assert.Equal(t, perm.Action, foundPerm.Action)
	}

	// Delete one permission
	err := repo.Delete(context.TODO(), testTenantID, "users:write")
	assert.NoError(t, err)

	// Verify only that one is deleted
	_, err = repo.Read(context.TODO(), testTenantID, "users:write")
	assert.Error(t, err)

	// Others should still exist
	_, err = repo.Read(context.TODO(), testTenantID, "users:read")
	assert.NoError(t, err)
	_, err = repo.Read(context.TODO(), testTenantID, "users:delete")
	assert.NoError(t, err)
}

func TestMongoDBPermissionsRepository_TenantIsolation(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	tenant1 := "00000000-0000-0000-0000-000000000001"
	tenant2 := "00000000-0000-0000-0000-000000000002"

	// Create same permission key in two different tenants
	perm1 := Permission{TenantID: tenant1, Key: "users:read", Name: "users", Action: "read"}
	err := repo.Create(context.TODO(), tenant1, perm1)
	assert.NoError(t, err)

	perm2 := Permission{TenantID: tenant2, Key: "users:read", Name: "users", Action: "read"}
	err = repo.Create(context.TODO(), tenant2, perm2)
	assert.NoError(t, err)

	// Verify permissions are isolated by tenant
	foundPerm1, err := repo.Read(context.TODO(), tenant1, "users:read")
	assert.NoError(t, err)
	assert.Equal(t, tenant1, foundPerm1.TenantID)

	foundPerm2, err := repo.Read(context.TODO(), tenant2, "users:read")
	assert.NoError(t, err)
	assert.Equal(t, tenant2, foundPerm2.TenantID)

	// Verify ReadAll returns only permissions for the specified tenant
	perms1, err := repo.ReadAll(context.TODO(), tenant1, shared.NewPageOptions(0, 0, ""))
	assert.NoError(t, err)
	assert.Len(t, perms1, 1)
	assert.Equal(t, tenant1, perms1[0].TenantID)

	perms2, err := repo.ReadAll(context.TODO(), tenant2, shared.NewPageOptions(0, 0, ""))
	assert.NoError(t, err)
	assert.Len(t, perms2, 1)
	assert.Equal(t, tenant2, perms2[0].TenantID)

	// Delete from one tenant shouldn't affect another
	err = repo.Delete(context.TODO(), tenant1, "users:read")
	assert.NoError(t, err)

	// Tenant1's permission is gone
	_, err = repo.Read(context.TODO(), tenant1, "users:read")
	assert.Error(t, err)

	// Tenant2's permission still exists
	_, err = repo.Read(context.TODO(), tenant2, "users:read")
	assert.NoError(t, err)
}
