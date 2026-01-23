package tenants

import (
	"context"
	"errors"

	"github.com/latebit-io/bulwarkauthadmin/internal/rbac"
)

const (
	tenantAdminRole            = "tenant_admin"
	tenantAdminRoleDescription = "tenant administrator"
	accountsManagePermission   = "accounts"
	accountsManageAction       = "manage"
	rbacManagePermission       = "rbac"
	rbacManageAction           = "manage"
)

type TenantAdminService interface {
	CreateTenantAdminRole(ctx context.Context, tenantID string) error
}

type TenantAdminServiceDefault struct {
	rolesRepository       rbac.RolesRepository
	permissionsRepository rbac.PermissionsRepository
}

// CreateTenantAdminRole creates the default tenant_admin role and associated permissions for a tenant
// This is idempotent - if the role already exists, it will not be created again
func (s *TenantAdminServiceDefault) CreateTenantAdminRole(ctx context.Context, tenantID string) error {
	// Create accounts:manage permission
	accountsPermission := rbac.NewPermission(tenantID, accountsManagePermission, accountsManageAction)
	err := s.permissionsRepository.Create(ctx, tenantID, accountsPermission)
	if err != nil {
		var duplicatePermission rbac.PermissionDuplicateError
		if !errors.As(err, &duplicatePermission) {
			return err
		}
	}

	// Create rbac:manage permission
	rbacPermission := rbac.NewPermission(tenantID, rbacManagePermission, rbacManageAction)
	err = s.permissionsRepository.Create(ctx, tenantID, rbacPermission)
	if err != nil {
		var duplicatePermission rbac.PermissionDuplicateError
		if !errors.As(err, &duplicatePermission) {
			return err
		}
	}

	// Create tenant_admin role with both permissions
	adminRole := rbac.NewRole(tenantID, tenantAdminRole, tenantAdminRoleDescription)
	adminRole.AddPermission(accountsPermission.Key)
	adminRole.AddPermission(rbacPermission.Key)
	err = s.rolesRepository.Create(ctx, tenantID, adminRole)
	if err != nil {
		var duplicateRole rbac.RoleDuplicateError
		if !errors.As(err, &duplicateRole) {
			return err
		}
	}

	return nil
}

func NewTenantAdminServiceDefault(rolesRepository rbac.RolesRepository, permissionsRepository rbac.PermissionsRepository) TenantAdminService {
	return &TenantAdminServiceDefault{
		rolesRepository:       rolesRepository,
		permissionsRepository: permissionsRepository,
	}
}
