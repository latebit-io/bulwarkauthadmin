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
			name: "Permission Already Exists",
			setupFunc: func(ctx context.Context, permRepo rbac.PermissionsRepository, roleRepo rbac.RolesRepository) error {
				adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
				return permRepo.Create(ctx, adminPermission)
			},
			expectedErr:   true,
			errorContains: "already exists",
		},
		{
			name: "Role Already Exists",
			setupFunc: func(ctx context.Context, permRepo rbac.PermissionsRepository, roleRepo rbac.RolesRepository) error {
				adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
				if err := permRepo.Create(ctx, adminPermission); err != nil {
					return err
				}
				adminRole := rbac.NewRole(bulwarkAdminRole, bulwarkAdminRoleDescription)
				return roleRepo.Create(ctx, adminRole)
			},
			expectedErr:   true,
			errorContains: "already exists",
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
				permission, err := permissionsRepo.Read(context.TODO(), bulwarkAdminPermission+":"+bulwarkAdminAction)
				assert.NoError(t, err)
				assert.NotNil(t, permission)
				assert.Equal(t, bulwarkAdminPermission, permission.Name)
				assert.Equal(t, bulwarkAdminAction, permission.Action)

				// Verify role was created
				role, err := rolesRepo.Read(context.TODO(), bulwarkAdminRole)
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
		// 		adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
		// 		if err := permRepo.Create(ctx, adminPermission); err != nil {
		// 			return err
		// 		}
		// 		adminRole := rbac.NewRole(bulwarkAdminRole, bulwarkAdminRoleDescription)
		// 		adminRole.AddPermission(adminPermission.Key)
		// 		return rolesRepo.Create(ctx, adminRole)
		// 	},
		// 	expectedErr: false,
		// 	validate: func(t *testing.T, ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository) {
		// 		account, err := accountRepo.ReadByEmail(ctx, "admin@latebit.io")
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
				adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
				if err := permRepo.Create(ctx, adminPermission); err != nil {
					return err
				}
				adminRole := rbac.NewRole(bulwarkAdminRole, bulwarkAdminRoleDescription)
				adminRole.AddPermission(adminPermission.Key)
				return rolesRepo.Create(ctx, adminRole)
			},
			expectedErr:   true,
			errorContains: "admin account requires password to be created",
		},
		// Note: This test requires bulwark-auth service running, skipped in unit tests
		// {
		// 	name:     "Assign Admin Role to Existing Account",
		// 	email:    "existing@latebit.io",
		// 	password: "SecurePassword123!",
		// 	setupFunc: func(ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository, permRepo rbac.PermissionsRepository, guard *bulwark.Guard) error {
		// 		// Create existing account in bulwark
		// 		if err := guard.Account.Create(ctx, "existing@latebit.io", "ExistingPassword123!"); err != nil {
		// 			return err
		// 		}

		// 		// Create internal roles
		// 		adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
		// 		if err := permRepo.Create(ctx, adminPermission); err != nil {
		// 			return err
		// 		}
		// 		adminRole := rbac.NewRole(bulwarkAdminRole, bulwarkAdminRoleDescription)
		// 		adminRole.AddPermission(adminPermission.Key)
		// 		return rolesRepo.Create(ctx, adminRole)
		// 	},
		// 	expectedErr: false,
		// 	validate: func(t *testing.T, ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository) {
		// 		account, err := accountRepo.ReadByEmail(ctx, "existing@latebit.io")
		// 		assert.NoError(t, err)
		// 		assert.Contains(t, account.Roles, bulwarkAdminRole)
		// 	},
		// },
		// Note: This test would require bulwark-auth to create the account first
		// The error happens after trying to call guard.Account.Create which needs the service
		// {
		// 	name:     "Admin Role Does Not Exist",
		// 	email:    "newadmin@latebit.io",
		// 	password: "SecurePassword123!",
		// 	setupFunc: func(ctx context.Context, accountRepo accounts.AccountRepository, rolesRepo rbac.RolesRepository, permRepo rbac.PermissionsRepository, guard *bulwark.Guard) error {
		// 		// Don't create the admin role
		// 		return nil
		// 	},
		// 	expectedErr:   true,
		// 	errorContains: "not found",
		// },
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

// Note: This test requires bulwark-auth service running, skipped in unit tests
// func TestAdminAccountsService_RegisterAccount_IdempotentRoleAssignment(t *testing.T) {
// 	client, db := setupMongoServer(t)
// 	defer cleanupMongoServer(t, client)

// 	accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)

// 	// Create internal roles
// 	adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
// 	err := permissionsRepo.Create(context.TODO(), adminPermission)
// 	assert.NoError(t, err)

// 	adminRole := rbac.NewRole(bulwarkAdminRole, bulwarkAdminRoleDescription)
// 	adminRole.AddPermission(adminPermission.Key)
// 	err = rolesRepo.Create(context.TODO(), adminRole)
// 	assert.NoError(t, err)

// 	service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

// 	// Register admin account first time
// 	err = service.RegisterAccount(context.TODO(), "admin@latebit.io", "SecurePassword123!")
// 	assert.NoError(t, err)

// 	account, err := accountsRepo.ReadByEmail(context.TODO(), "admin@latebit.io")
// 	assert.NoError(t, err)
// 	assert.Len(t, account.Roles, 1)
// 	assert.Contains(t, account.Roles, bulwarkAdminRole)

// 	// Register same admin account again (should be idempotent)
// 	err = service.RegisterAccount(context.TODO(), "admin@latebit.io", "SecurePassword123!")
// 	assert.NoError(t, err)

// 	// Verify role is still assigned only once
// 	account, err = accountsRepo.ReadByEmail(context.TODO(), "admin@latebit.io")
// 	assert.NoError(t, err)
// 	assert.Len(t, account.Roles, 1)
// 	assert.Contains(t, account.Roles, bulwarkAdminRole)
// }

// Note: This test requires bulwark-auth service running, skipped in unit tests
// func TestAdminAccountsService_RegisterAccount_AccountFlags(t *testing.T) {
// 	client, db := setupMongoServer(t)
// 	defer cleanupMongoServer(t, client)

// 	accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)

// 	// Create internal roles
// 	adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
// 	err := permissionsRepo.Create(context.TODO(), adminPermission)
// 	assert.NoError(t, err)

// 	adminRole := rbac.NewRole(bulwarkAdminRole, bulwarkAdminRoleDescription)
// 	adminRole.AddPermission(adminPermission.Key)
// 	err = rolesRepo.Create(context.TODO(), adminRole)
// 	assert.NoError(t, err)

// 	service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

// 	// Register admin account
// 	err = service.RegisterAccount(context.TODO(), "admin@latebit.io", "SecurePassword123!")
// 	assert.NoError(t, err)

// 	// Verify account flags are set correctly
// 	account, err := accountsRepo.ReadByEmail(context.TODO(), "admin@latebit.io")
// 	assert.NoError(t, err)
// 	assert.True(t, account.IsVerified, "Admin account should be verified")
// 	assert.True(t, account.IsEnabled, "Admin account should be enabled")
// 	assert.False(t, account.IsDeleted, "Admin account should not be deleted")
// }

// Note: This test requires bulwark-auth service running, skipped in unit tests
// func TestAdminAccountsService_Integration(t *testing.T) {
// 	client, db := setupMongoServer(t)
// 	defer cleanupMongoServer(t, client)

// 	accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)

// 	service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

// 	// Step 1: Create internal roles
// 	err := service.CreateInternalRoles(context.TODO())
// 	assert.NoError(t, err)

// 	// Verify permission exists
// 	permission, err := permissionsRepo.Read(context.TODO(), bulwarkAdminPermission+":"+bulwarkAdminAction)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, permission)

// 	// Verify role exists
// 	role, err := rolesRepo.Read(context.TODO(), bulwarkAdminRole)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, role)
// 	assert.Contains(t, role.Permissions, permission.Key)

// 	// Step 2: Register admin account
// 	err = service.RegisterAccount(context.TODO(), "admin@latebit.io", "SecurePassword123!")
// 	assert.NoError(t, err)

// 	// Verify account was created and configured correctly
// 	account, err := accountsRepo.ReadByEmail(context.TODO(), "admin@latebit.io")
// 	assert.NoError(t, err)
// 	assert.Equal(t, "admin@latebit.io", account.Email)
// 	assert.True(t, account.IsVerified)
// 	assert.True(t, account.IsEnabled)
// 	assert.False(t, account.IsDeleted)
// 	assert.Contains(t, account.Roles, bulwarkAdminRole)

// 	// Step 3: Verify can register multiple times without error
// 	err = service.RegisterAccount(context.TODO(), "admin@latebit.io", "SecurePassword123!")
// 	assert.NoError(t, err)
// }

func TestNewAdminAccountsServiceDefault(t *testing.T) {
	client, db := setupMongoServer(t)
	defer cleanupMongoServer(t, client)

	accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)

	service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

	assert.NotNil(t, service)
	assert.Implements(t, (*AdminAccountsService)(nil), service)
}

// Note: Error handling tests require bulwark-auth service, skipped in unit tests
// func TestAdminAccountsService_ErrorHandling(t *testing.T) {
// 	tests := []struct {
// 		name        string
// 		testFunc    func(t *testing.T, service AdminAccountsService)
// 		description string
// 	}{
// 		{
// 			name: "RegisterAccount handles missing role gracefully",
// 			testFunc: func(t *testing.T, service AdminAccountsService) {
// 				err := service.RegisterAccount(context.TODO(), "test@example.com", "password123")
// 				assert.Error(t, err)
// 				var roleNotFound rbac.RoleNotFoundError
// 				assert.True(t, errors.As(err, &roleNotFound))
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			client, db := setupMongoServer(t)
// 			defer cleanupMongoServer(t, client)

// 			accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard := setupTestServices(t, db)
// 			service := NewAdminAccountsServiceDefault(accountsRepo, rolesRepo, permissionsRepo, rbacAccountService, guard)

// 			tt.testFunc(t, service)
// 		})
// 	}
// }

func TestAdminAccountsService_Constants(t *testing.T) {
	// Verify constants are set correctly
	assert.Equal(t, "bulwark_admin", bulwarkAdminRole)
	assert.Equal(t, "bulwark internal admin", bulwarkAdminRoleDescription)
	assert.Equal(t, "bulwark_admin", bulwarkAdminPermission)
	assert.Equal(t, "write", bulwarkAdminAction)
}
