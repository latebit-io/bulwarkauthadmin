package rbac

import (
	"context"
	"slices"

	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
	"github.com/latebit-io/bulwarkauthadmin/internal/rbac"
)

type AccountRBACService interface {
	AssignRole(ctx context.Context, tenantID, accountID, role string) error
	RemoveRole(ctx context.Context, tenantID, accountID, role string) error
	AssignPermission(ctx context.Context, tenantID, accountID, permission string) error
	RemovePermission(ctx context.Context, tenantID, accountID, permission string) error
}

type AccountRBACServiceDefault struct {
	accountRepo       accounts.AccountRepository
	permissionService rbac.PermissionService
	rolesService      rbac.RoleService
}

// AssignPermission implements AccountRBACService.
func (a *AccountRBACServiceDefault) AssignPermission(ctx context.Context, tenantID, accountID string, permission string) error {
	exists, err := a.permissionService.DoesPermissionExist(ctx, tenantID, permission)
	if err != nil {
		return err
	}
	if !exists {
		return rbac.PermissionNotFoundError{
			Value: permission,
		}
	}
	account, err := a.accountRepo.ReadById(ctx, tenantID, accountID)
	if err != nil {
		return err
	}
	if account.Permissions == nil {
		account.Permissions = make([]string, 0)
	} else if slices.Contains(account.Permissions, permission) {
		return nil
	}

	account.Permissions = append(account.Permissions, permission)
	return a.accountRepo.Update(ctx, tenantID, *account)
}

// AssignRole implements AccountRBACService.
func (a *AccountRBACServiceDefault) AssignRole(ctx context.Context, tenantID, accountID string, role string) error {
	_, err := a.rolesService.GetRole(ctx, tenantID, role)
	if err != nil {
		return err
	}
	account, err := a.accountRepo.ReadById(ctx, tenantID, accountID)
	if err != nil {
		return err
	}
	if account.Roles == nil {
		account.Roles = make([]string, 0)
	} else if slices.Contains(account.Roles, role) {
		return nil
	}

	account.Roles = append(account.Roles, role)
	return a.accountRepo.Update(ctx, tenantID, *account)
}

// RemovePermission implements AccountRBACService.
func (a *AccountRBACServiceDefault) RemovePermission(ctx context.Context, tenantID, accountID string, permission string) error {
	account, err := a.accountRepo.ReadById(ctx, tenantID, accountID)
	if err != nil {
		return err
	}
	if account.Permissions == nil {
		return nil
	}
	if !slices.Contains(account.Permissions, permission) {
		return nil
	}
	account.Permissions = slices.DeleteFunc(account.Permissions, func(p string) bool { return p == permission })
	return a.accountRepo.Update(ctx, tenantID, *account)
}

// RemoveRole implements AccountRBACService.
func (a *AccountRBACServiceDefault) RemoveRole(ctx context.Context, tenantID, accountID string, role string) error {
	account, err := a.accountRepo.ReadById(ctx, tenantID, accountID)
	if err != nil {
		return err
	}
	if account.Roles == nil {
		return nil
	}
	if !slices.Contains(account.Roles, role) {
		return nil
	}

	account.Roles = slices.DeleteFunc(account.Roles, func(r string) bool { return r == role })
	return a.accountRepo.Update(ctx, tenantID, *account)
}

func NewAccountRBACServiceDefault(accountRepo accounts.AccountRepository, permissionService rbac.PermissionService,
	rolesService rbac.RoleService) AccountRBACService {
	return &AccountRBACServiceDefault{
		accountRepo:       accountRepo,
		permissionService: permissionService,
		rolesService:      rolesService,
	}
}
