package admin

import (
	"context"
	"errors"

	bulwark "github.com/latebit-io/bulwark-auth-guard"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
	rbacAccounts "github.com/latebit-io/bulwarkauthadmin/internal/accounts/rbac"
	"github.com/latebit-io/bulwarkauthadmin/internal/rbac"
)

const (
	bulwarkAdminRole            = "bulwark_admin"
	bulwarkAdminRoleDescription = "bulwark internal admin"
	bulwarkAdminPermission      = "bulwark_admin"
	bulwarkAdminAction          = "write"
)

type AdminAccountsService interface {
	RegisterAccount(ctx context.Context, tenantID, email string, password string) error
	CreateInternalRoles(ctx context.Context) error
}

type AdminAccountsServiceDefault struct {
	accountsRepository    accounts.AccountRepository
	rolesRepository       rbac.RolesRepository
	permissionsRepository rbac.PermissionsRepository
	rbacAccountService    rbacAccounts.AccountRBACService

	auth *bulwark.Guard
}

// CreateInternalRoles implements AdminAccountsService.
func (a *AdminAccountsServiceDefault) CreateInternalRoles(ctx context.Context) error {
	adminPermission := rbac.NewPermission(bulwarkAdminPermission, bulwarkAdminAction)
	err := a.permissionsRepository.Create(ctx, adminPermission)
	if err != nil {
		var duplicatePermission rbac.PermissionDuplicateError
		if !errors.As(err, &duplicatePermission) {
			return err
		}
	}

	adminRole := rbac.NewRole(bulwarkAdminRole, bulwarkAdminRoleDescription)
	adminRole.AddPermission(adminPermission.Key)
	err = a.rolesRepository.Create(ctx, adminRole)
	if err != nil {
		var duplicateRole rbac.RoleDuplicateError
		if !errors.As(err, &duplicateRole) {
			return err
		}
	}
	return nil
}

// RegisterAccount implements AdminAccountsService.
func (a *AdminAccountsServiceDefault) RegisterAccount(ctx context.Context, email string, password string) error {
	admin, err := a.accountsRepository.ReadByEmail(ctx, email)
	if err != nil {
		var accountNotFound accounts.AccountNotFoundError
		if !errors.As(err, &accountNotFound) {
			return err
		}

		if password == "" {
			return errors.New("admin account requires password to be created")
		}

		err = a.auth.Account.Create(ctx, email, password)
		if err != nil {
			var duplicateAccount accounts.AccountDuplicateError
			if !errors.As(err, &duplicateAccount) {
				return err
			}
		}
		admin, err = a.accountsRepository.ReadByEmail(ctx, email)
		if err != nil {
			return err
		}
		admin.IsVerified = true
		admin.IsEnabled = true
		admin.IsDeleted = false

		err = a.accountsRepository.Update(ctx, *admin)
		if err != nil {
			return err
		}
	}

	err = a.rbacAccountService.AssignRole(ctx, admin.ID, bulwarkAdminRole)
	if err != nil {
		return err
	}
	return nil
}

func NewAdminAccountsServiceDefault(accountsRepository accounts.AccountRepository, rolesRepository rbac.RolesRepository, permissionsRepository rbac.PermissionsRepository, rbacAccountService rbacAccounts.AccountRBACService, auth *bulwark.Guard) AdminAccountsService {
	return &AdminAccountsServiceDefault{
		accountsRepository:    accountsRepository,
		permissionsRepository: permissionsRepository,
		auth:                  auth,
		rolesRepository:       rolesRepository,
		rbacAccountService:    rbacAccountService,
	}
}
