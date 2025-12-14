package rbac

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
			role:        NewRole("admin", "Administrator with full access"),
			expectedErr: nil,
		},
		{
			name: "Valid Role with Permissions",
			role: Role{
				ID:          "role-1",
				Name:        "moderator",
				Description: "Moderator role",
				Permissions: []string{"users:read", "posts:edit"},
			},
			expectedErr: nil,
		},
		{
			name:        "Duplicate Role",
			role:        NewRole("admin", "Administrator with full access"),
			expectedErr: RoleDuplicateError{Value: "admin"},
		},
	}

	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	repo := NewMongoDBRolesRepository(db)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(context.TODO(), tt.role)

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
			createRole:       &Role{ID: "role-1", Name: "admin", Description: "Administrator"},
			expectedRoleName: "admin",
			expectedErr:      false,
		},
		{
			name:             "Role with Permissions",
			roleName:         "moderator",
			createRole:       &Role{ID: "role-2", Name: "moderator", Description: "Moderator", Permissions: []string{"users:read"}},
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
				err := repo.Create(context.TODO(), *tt.createRole)
				assert.NoError(t, err)
			}

			role, err := repo.Read(context.TODO(), tt.roleName)

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
			initialRole: NewRole("admin", "Administrator"),
			updatedRole: Role{
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
			initialRole: NewRole("editor", "Content Editor"),
			updatedRole: Role{
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
				err := repo.Create(context.TODO(), tt.initialRole)
				assert.NoError(t, err)
			}

			err := repo.Update(context.TODO(), tt.updatedRole)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.initialRole.Name != "" {
					updatedRole, err := repo.Read(context.TODO(), tt.updatedRole.Name)
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedDescription, updatedRole.Description)
					assert.Equal(t, tt.expectedPermissions, updatedRole.Permissions)
				}
			}
		})
	}
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
			createRole:  &Role{ID: "role-1", Name: "admin", Description: "Administrator"},
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
				err := repo.Create(context.TODO(), *tt.createRole)
				assert.NoError(t, err)
			}

			err := repo.Delete(context.TODO(), tt.roleName)

			if tt.expectedErr {
				assert.Error(t, err)
				var notFoundErr RoleNotFoundError
				assert.True(t, errors.As(err, &notFoundErr))
			} else {
				assert.NoError(t, err)

				// Verify role is deleted
				_, err := repo.Read(context.TODO(), tt.roleName)
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
	role := NewRole("editor", "Content Editor")
	err := repo.Create(context.TODO(), role)
	assert.NoError(t, err)

	// Add permission
	role.AddPermission("posts:create")
	role.AddPermission("posts:edit")
	err = repo.Update(context.TODO(), role)
	assert.NoError(t, err)

	// Verify permissions added
	updatedRole, err := repo.Read(context.TODO(), "editor")
	assert.NoError(t, err)
	assert.Len(t, updatedRole.Permissions, 2)
	assert.Contains(t, updatedRole.Permissions, "posts:create")
	assert.Contains(t, updatedRole.Permissions, "posts:edit")

	// Remove permission
	updatedRole.RemovePermission("posts:create")
	err = repo.Update(context.TODO(), *updatedRole)
	assert.NoError(t, err)

	// Verify permission removed
	finalRole, err := repo.Read(context.TODO(), "editor")
	assert.NoError(t, err)
	assert.Len(t, finalRole.Permissions, 1)
	assert.Contains(t, finalRole.Permissions, "posts:edit")
	assert.NotContains(t, finalRole.Permissions, "posts:create")
}
