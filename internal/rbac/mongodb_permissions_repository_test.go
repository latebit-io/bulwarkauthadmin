package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
	"github.com/stretchr/testify/assert"
)

func TestMongoDBPermissionsRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		permission  Permission
		expectedErr error
	}{
		{
			name:        "Valid Permission",
			permission:  NewPermission("users", "read"),
			expectedErr: nil,
		},
		{
			name:        "Valid Permission with Action",
			permission:  NewPermission("posts", "create"),
			expectedErr: nil,
		},
		{
			name: "Duplicate Permission",
			permission: Permission{
				ID:     "perm-1",
				Key:    "users:read",
				Name:   "users",
				Action: "read",
			},
			expectedErr: PermissionDuplicateError{Value: "users:read"},
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	// Create first permission for duplicate test
	firstPerm := Permission{
		ID:     "perm-1",
		Key:    "users:read",
		Name:   "users",
		Action: "read",
	}
	err := repo.Create(context.TODO(), firstPerm)
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(context.TODO(), tt.permission)

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
				ID:     "perm-1",
				Key:    "users:read",
				Name:   "users",
				Action: "read",
			},
			expectedName:   "users",
			expectedAction: "read",
			expectedErr:    false,
		},
		{
			name:          "Permission with Different Action",
			permissionKey: "posts:delete",
			createPermission: &Permission{
				ID:     "perm-2",
				Key:    "posts:delete",
				Name:   "posts",
				Action: "delete",
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
				err := repo.Create(context.TODO(), *tt.createPermission)
				assert.NoError(t, err)
			}

			permission, err := repo.Read(context.TODO(), tt.permissionKey)

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
				{ID: "perm-1", Key: "users:read", Name: "users", Action: "read"},
				{ID: "perm-2", Key: "users:write", Name: "users", Action: "write"},
				{ID: "perm-3", Key: "posts:delete", Name: "posts", Action: "delete"},
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
				{ID: "perm-1", Key: "users:read", Name: "users", Action: "read"},
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
				err := repo.Create(context.TODO(), perm)
				assert.NoError(t, err)
			}

			permissions, err := repo.ReadAll(context.TODO(), shared.NewPageOptions(0, 0, ""))

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
				ID:     "perm-1",
				Key:    "users:read",
				Name:   "users",
				Action: "read",
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
				err := repo.Create(context.TODO(), *tt.createPermission)
				assert.NoError(t, err)
			}

			err := repo.Delete(context.TODO(), tt.permissionKey)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr PermissionNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)

				// Verify permission is deleted
				_, err := repo.Read(context.TODO(), tt.permissionKey)
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
		{ID: "perm-1", Key: "users:read", Name: "users", Action: "read"},
		{ID: "perm-2", Key: "users:write", Name: "users", Action: "write"},
		{ID: "perm-3", Key: "users:delete", Name: "users", Action: "delete"},
		{ID: "perm-4", Key: "posts:read", Name: "posts", Action: "read"},
		{ID: "perm-5", Key: "posts:write", Name: "posts", Action: "write"},
	}

	for _, perm := range permissions {
		err := repo.Create(context.TODO(), perm)
		assert.NoError(t, err)
	}

	// Test ReadAll returns all
	allPerms, err := repo.ReadAll(context.TODO(), shared.NewPageOptions(0, 0, ""))
	assert.NoError(t, err)
	assert.Equal(t, 5, len(allPerms))
}

func TestMongoDBPermissionsRepository_MultipleActionsOnSameResource(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBPermissionsRepository(db)

	// Create multiple actions on same resource
	permissions := []Permission{
		{ID: "perm-1", Key: "users:read", Name: "users", Action: "read"},
		{ID: "perm-2", Key: "users:write", Name: "users", Action: "write"},
		{ID: "perm-3", Key: "users:delete", Name: "users", Action: "delete"},
	}

	for _, perm := range permissions {
		err := repo.Create(context.TODO(), perm)
		assert.NoError(t, err)
	}

	// Verify all permissions exist
	for _, perm := range permissions {
		foundPerm, err := repo.Read(context.TODO(), perm.Key)
		assert.NoError(t, err)
		assert.Equal(t, perm.Name, foundPerm.Name)
		assert.Equal(t, perm.Action, foundPerm.Action)
	}

	// Delete one permission
	err := repo.Delete(context.TODO(), "users:write")
	assert.NoError(t, err)

	// Verify only that one is deleted
	_, err = repo.Read(context.TODO(), "users:write")
	assert.Error(t, err)

	// Others should still exist
	_, err = repo.Read(context.TODO(), "users:read")
	assert.NoError(t, err)
	_, err = repo.Read(context.TODO(), "users:delete")
	assert.NoError(t, err)
}
