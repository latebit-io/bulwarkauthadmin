package rbac

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
)

type Role struct {
	ID          string    `json:"id" bson:"id"`
	TenantID    string    `json:"tenantId" bson:"tenantId"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	Permissions []string  `json:"permissionIds" bson:"permissionIds"`
	Created     time.Time `json:"created" bson:"created"`
	Modified    time.Time `json:"modified" bson:"modified"`
}

func NewRole(tenantID, name, description string) Role {
	return Role{
		ID:          uuid.New().String(),
		TenantID:    tenantID,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		Permissions: []string{},
		Created:     time.Now(),
		Modified:    time.Now(),
	}
}

func (r *Role) RemovePermission(permissionName string) {
	for i, permission := range r.Permissions {
		if permission == permissionName {
			r.Permissions = append(r.Permissions[:i], r.Permissions[i+1:]...)
			break
		}
	}
}

func (r *Role) AddPermission(permissionName string) {
	permissionName = strings.TrimSpace(permissionName)
	if slices.Contains(r.Permissions, permissionName) {
		return
	}
	r.Permissions = append(r.Permissions, permissionName)
}

type Permission struct {
	ID       string    `json:"id" bson:"id"`
	TenantID string    `json:"tenantId" bson:"tenantId"`
	Key      string    `json:"key" bson:"key"`
	Name     string    `json:"name" bson:"name"`
	Action   string    `json:"action" bson:"action"`
	Created  time.Time `json:"created" bson:"created"`
	Modified time.Time `json:"modified" bson:"modified"`
}

func NewPermission(tenantID, name, action string) Permission {
	name = strings.TrimSpace(name)
	action = strings.TrimSpace(action)
	return Permission{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Key:      fmt.Sprintf("%s:%s", name, action),
		Name:     name,
		Action:   action,
		Created:  time.Now(),
		Modified: time.Now(),
	}
}

type RolesRepository interface {
	Create(ctx context.Context, tenantID string, role Role) error
	Read(ctx context.Context, tenantID string, roleName string) (*Role, error)
	ReadAll(ctx context.Context, tenantID string, paging shared.PageOptions) ([]Role, error)
	Update(ctx context.Context, tenantID string, role Role) error
	Delete(ctx context.Context, tenantID string, roleName string) error
}

type PermissionsRepository interface {
	Create(ctx context.Context, tenantID string, permission Permission) error
	Read(ctx context.Context, tenantID string, key string) (*Permission, error)
	ReadAll(ctx context.Context, tenantID string, paging shared.PageOptions) ([]Permission, error)
	Delete(ctx context.Context, tenantID string, key string) error
}

type RoleService interface {
	CreateRole(ctx context.Context, tenantID string, name, description string) error
	UpdateRole(ctx context.Context, tenantID string, role, description string) error
	GetRole(ctx context.Context, tenantID string, name string) (Role, error)
	ListRoles(ctx context.Context, tenantID string, paging shared.PageOptions) ([]Role, error)
	AddPermission(ctx context.Context, tenantID string, role, permissionKey string) error
	RemovePermission(ctx context.Context, tenantID string, role, permissionKey string) error
	DeleteRole(ctx context.Context, tenantID string, role string) error
}

type RoleServiceDefault struct {
	roleRepository RolesRepository
}

// GetRole implements RoleService.
func (r *RoleServiceDefault) GetRole(ctx context.Context, tenantID string, name string) (Role, error) {
	role, err := r.roleRepository.Read(ctx, tenantID, name)
	if err != nil {
		return Role{}, err
	}
	return *role, nil
}

// ListRoles implements RoleService.
func (r *RoleServiceDefault) ListRoles(ctx context.Context, tenantID string, paging shared.PageOptions) ([]Role, error) {
	roles, err := r.roleRepository.ReadAll(ctx, tenantID, paging)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// AddPermission implements RoleService.
func (r *RoleServiceDefault) AddPermission(ctx context.Context, tenantID, roleName string, permissionKey string) error {
	role, err := r.roleRepository.Read(ctx, tenantID, roleName)
	if err != nil {
		return err
	}
	role.AddPermission(permissionKey)
	return r.roleRepository.Update(ctx, tenantID, *role)
}

// CreateRole implements RoleService.
func (r *RoleServiceDefault) CreateRole(ctx context.Context, tenantID, name string, description string) error {
	if err := r.roleRepository.Create(ctx, tenantID, NewRole(tenantID, name, description)); err != nil {
		return err
	}
	return nil
}

// DeleteRole implements RoleService.
func (r *RoleServiceDefault) DeleteRole(ctx context.Context, tenantID, roleName string) error {
	if err := r.roleRepository.Delete(ctx, tenantID, roleName); err != nil {
		return err
	}
	return nil
}

// RemovePermission implements RoleService.
func (r *RoleServiceDefault) RemovePermission(ctx context.Context, tenantID, roleName string, permissionKey string) error {
	role, err := r.roleRepository.Read(ctx, tenantID, roleName)
	if err != nil {
		return err
	}

	role.RemovePermission(permissionKey)
	if err := r.roleRepository.Update(ctx, tenantID, *role); err != nil {
		return err
	}
	return nil
}

// UpdateRole implements RoleService.
func (r *RoleServiceDefault) UpdateRole(ctx context.Context, tenantID, roleName string, description string) error {
	role, err := r.roleRepository.Read(ctx, tenantID, roleName)
	if err != nil {
		return err
	}

	role.Description = description
	if err := r.roleRepository.Update(ctx, tenantID, *role); err != nil {
		return err
	}
	return nil
}

func NewRoleServiceDefault(roleRepository RolesRepository) RoleService {
	return &RoleServiceDefault{
		roleRepository: roleRepository,
	}
}

type PermissionService interface {
	CreatePermission(ctx context.Context, tenantID, name string, action string) error
	DeletePermission(ctx context.Context, tenantID, name string, action string) error
	ListPermissions(ctx context.Context, tenantID string, paging shared.PageOptions) ([]Permission, error)
	DoesPermissionExist(ctx context.Context, tenantID, permissionKey string) (bool, error)
}

type PermissionServiceDefault struct {
	permissionRepository PermissionsRepository
}

// CreatePermission implements PermissionService.
func (p PermissionServiceDefault) CreatePermission(ctx context.Context, tenantID, name string, action string) error {
	newPermission := NewPermission(tenantID, name, action)
	if err := p.permissionRepository.Create(ctx, tenantID, newPermission); err != nil {
		return err
	}
	return nil
}

// DeletePermission implements PermissionService.
func (p PermissionServiceDefault) DeletePermission(ctx context.Context, tenantID, name string, action string) error {
	if err := p.permissionRepository.Delete(ctx, tenantID, fmt.Sprintf("%s:%s", name, action)); err != nil {
		return err
	}
	return nil
}

// DoesPermissionExist implements PermissionService.
func (p PermissionServiceDefault) DoesPermissionExist(ctx context.Context, tenantID, permissionKey string) (bool, error) {
	_, err := p.permissionRepository.Read(ctx, tenantID, permissionKey)
	if err != nil {
		var notFoundErr PermissionNotFoundError
		if errors.As(err, &notFoundErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListPermission implements PermissionService.
func (p PermissionServiceDefault) ListPermissions(ctx context.Context, tenantID string, paging shared.PageOptions) ([]Permission, error) {
	permissions, err := p.permissionRepository.ReadAll(ctx, tenantID, paging)
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

func NewPermissionServiceDefault(permissionRepository PermissionsRepository) PermissionService {
	return PermissionServiceDefault{
		permissionRepository: permissionRepository,
	}
}
