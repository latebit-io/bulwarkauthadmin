package admin

import (
	"context"
	"net/http"
	"testing"

	bulwark "github.com/latebit-io/bulwark-auth-guard"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
	rbacAccounts "github.com/latebit-io/bulwarkauthadmin/internal/accounts/rbac"
	"github.com/latebit-io/bulwarkauthadmin/internal/rbac"
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

func setupTestServices(t *testing.T, db *mongo.Database) (
	accounts.AccountRepository,
	rbac.RolesRepository,
	rbac.PermissionsRepository,
	rbacAccounts.AccountRBACService,
	*bulwark.Guard,
) {
	accountsRepo := accounts.NewMongoDBAccountRepository(db)
	rolesRepo := rbac.NewMongoDBRolesRepository(db)
	permissionsRepo := rbac.NewMongoDBPermissionsRepository(db)

	roleService := rbac.NewRoleServiceDefault(rolesRepo)
	permissionService := rbac.NewPermissionServiceDefault(permissionsRepo)
	rbacAccountService := rbacAccounts.NewAccountRBACServiceDefault(accountsRepo, permissionService, roleService)

	// Initialize bulwark guard with test URL (won't be called in most tests)
	httpClient := &http.Client{}
	guard := bulwark.NewGuard("http://localhost:8080", httpClient)

	return accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard
}

func TestAdminAccountsService_CreateInternalRoles(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(ctx context.Context, permRepo rbac.PermissionsRepository, roleRepo rbac.RolesRepository) error
		expectedErr   bool
		errorContains string
	}{
		{
			name:        "Create Admin Role and Permission Successfully",
			setupFunc:   nil,
			expectedErr: false,
		},
		{
			name: "Permission Already Exists - Should Be Idempotent",
			setupFunc: func(ctx context.Context, permRepo rbac.PermissionsRepository, roleRepo rbac.RolesRepository) error {
				adminPermission := rbac.NewPermission(systemTenantID, bulwarkAdminPermission, bulwarkAdminAction)
				return permRepo.Create(ctx, systemTenantID, adminPermission)
			},
			expectedErr: false, // CreateInternalRoles handles duplicates gracefully
		},
		{
			name: "Role Already Exists - Should Be Idempotent",
			setupFunc: func(ctx context.Context, permRepo rbac.PermissionsRepository, roleRepo rbac.RolesRepository) error {
				adminPermission := rbac.NewPermission(systemTenantID, bulwarkAdminPermission, bulwarkAdminAction)
				if err := permRepo.Create(ctx, systemTenantID, adminPermission); err != nil {
					return err
				}
				adminRole := rbac.NewRole(systemTenantID, bulwarkAdminRole, bulwarkAdminRoleDescription)
				return roleRepo.Create(ctx, systemTenantID, adminRole)
			},
			expectedErr: false, // CreateInternalRoles handles duplicates gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, db := setupMongoServer(t)
			defer cleanupMongoServer(t, client)

			accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)

			if tt.setupFunc != nil {
				err := tt.setupFunc(context.TODO(), permissionsRepo, rolesRepo)
				assert.NoError(t, err)
			}

			service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

			err := service.CreateInternalRoles(context.TODO())

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify permission was created
				permission, err := permissionsRepo.Read(context.TODO(), systemTenantID, bulwarkAdminPermission+":"+bulwarkAdminAction)
				assert.NoError(t, err)
				assert.NotNil(t, permission)
				assert.Equal(t, bulwarkAdminPermission, permission.Name)
				assert.Equal(t, bulwarkAdminAction, permission.Action)

				// Verify role was created
				role, err := rolesRepo.Read(context.TODO(), systemTenantID, bulwarkAdminRole)
				assert.NoError(t, err)
				assert.NotNil(t, role)
				assert.Equal(t, bulwarkAdminRole, role.Name)
				assert.Equal(t, bulwarkAdminRoleDescription, role.Description)
				assert.Contains(t, role.Permissions, bulwarkAdminPermission+":"+bulwarkAdminAction)
			}
		})
	}
}

func TestAdminAccountsService_RegisterAccount(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		setupFunc     func(ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository, permRepo rbac.PermissionsRepository, guard *bulwark.Guard) error
		expectedErr   bool
		errorContains string
		validate      func(t *testing.T, ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository)
	}{
		// Note: This test requires bulwark-auth service running, skipped in unit tests
		// {
		// 	name:     "Register New Admin Account",
		// 	email:    "admin@latebit.io",
		// 	password: "SecurePassword123!",
		// 	setupFunc: func(ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository, permRepo rbac.PermissionsRepository, guard *bulwark.Guard) error {
		// 		// Create internal roles first
		// 		adminPermission := rbac.NewPermission(systemTenantID, bulwarkAdminPermission, bulwarkAdminAction)
		// 		if err := permRepo.Create(ctx, systemTenantID, adminPermission); err != nil {
		// 			return err
		// 		}
		// 		adminRole := rbac.NewRole(systemTenantID, bulwarkAdminRole, bulwarkAdminRoleDescription)
		// 		adminRole.AddPermission(adminPermission.Key)
		// 		return rolesRepo.Create(ctx, systemTenantID, adminRole)
		// 	},
		// 	expectedErr: false,
		// 	validate: func(t *testing.T, ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository) {
		// 		account, err := accountRepo.ReadByEmail(ctx, systemTenantID, "admin@latebit.io")
		// 		assert.NoError(t, err)
		// 		assert.True(t, account.IsVerified)
		// 		assert.True(t, account.IsEnabled)
		// 		assert.False(t, account.IsDeleted)
		// 		assert.Contains(t, account.Roles, bulwarkAdminRole)
		// 	},
		// },
		{
			name:     "Register Admin Account Without Password",
			email:    "admin@latebit.io",
			password: "",
			setupFunc: func(ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository, permRepo rbac.PermissionsRepository, guard *bulwark.Guard) error {
				adminPermission := rbac.NewPermission(systemTenantID, bulwarkAdminPermission, bulwarkAdminAction)
				if err := permRepo.Create(ctx, systemTenantID, adminPermission); err != nil {
					return err
				}
				adminRole := rbac.NewRole(systemTenantID, bulwarkAdminRole, bulwarkAdminRoleDescription)
				adminRole.AddPermission(adminPermission.Key)
				return rolesRepo.Create(ctx, systemTenantID, adminRole)
			},
			expectedErr:   true,
			errorContains: "admin account requires password to be created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, db := setupMongoServer(t)
			defer cleanupMongoServer(t, client)

			accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)

			if tt.setupFunc != nil {
				err := tt.setupFunc(context.TODO(), accountsRepo, rolesRepo, permissionsRepo, guard)
				assert.NoError(t, err)
			}

			service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

			err := service.RegisterAccount(context.TODO(), tt.email, tt.password)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, context.TODO(), accountsRepo, rolesRepo)
				}
			}
		})
	}
}

func TestNewAdminAccountsServiceDefault(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)

	service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

	assert.NotNil(t, service)
	assert.Implements(t, (*AdminAccountsService)(nil), service)
}

func TestAdminAccountsService_Constants(t *testing.T) {
	// Verify constants are set correctly
	assert.Equal(t, "bulwark_admin", bulwarkAdminRole)
	assert.Equal(t, "bulwark internal admin", bulwarkAdminRoleDescription)
	assert.Equal(t, "bulwark_admin", bulwarkAdminPermission)
	assert.Equal(t, "write", bulwarkAdminAction)
}
