package rbac

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
)

type Role struct {
	ID          string    `json:"id" bson:"id"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	Permissions []string  `json:"permissionIds" bson:"permissionIds"`
	Created     time.Time `json:"created" bson:"created"`
	Modified    time.Time `json:"modified" bson:"modified"`
}

func NewRole(name, description string) Role {
	return Role{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
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
	r.Permissions = append(r.Permissions, permissionName)
}

type Permission struct {
	ID       string    `json:"id" bson:"id"`
	Key      string    `json:"key" bson:"key"`
	Name     string    `json:"name" bson:"name"`
	Action   string    `json:"action" bson:"action"`
	Created  time.Time `json:"created" bson:"created"`
	Modified time.Time `json:"modified" bson:"modified"`
}

func NewPermission(name, action string) Permission {
	return Permission{
		ID:       uuid.New().String(),
		Key:      fmt.Sprintf("%s:%s", name, uuid.New().String()),
		Name:     name,
		Action:   action,
		Created:  time.Now(),
		Modified: time.Now(),
	}
}

type RolesRepository interface {
	Create(ctx context.Context, role Role) error
	Read(ctx context.Context, roleName string) (*Role, error)
	Update(ctx context.Context, role Role) error
	Delete(ctx context.Context, roleName string) error
}

type PermissionsRepository interface {
	Create(ctx context.Context, permission Permission) error
	Read(ctx context.Context, key string) (*Permission, error)
	ReadAll(ctx context.Context, paging shared.PageOptions) ([]Permission, error)
	Delete(ctx context.Context, key string) error
}

type RoleService interface {
	CreateRole(ctx context.Context, name, description string) error
	UpdateRole(ctx context.Context, role, description string) error
	AddPermission(ctx context.Context, role, permissionKey string) error
	RemovePermission(ctx context.Context, role, permissionKey string) error
	DeleteRole(ctx context.Context, role string) error
}

type RoleServiceDefault struct {
	roleRepository RolesRepository
}

// AddPermission implements RoleService.
func (r *RoleServiceDefault) AddPermission(ctx context.Context, roleName string, permissionKey string) error {
	role, err := r.roleRepository.Read(ctx, roleName)
	if err != nil {
		return err
	}
	role.AddPermission(permissionKey)
	return r.roleRepository.Update(ctx, *role)
}

// CreateRole implements RoleService.
func (r *RoleServiceDefault) CreateRole(ctx context.Context, name string, description string) error {
	newRole := Role{
		ID:       uuid.New().String(),
		Name:     name,
		Modified: time.Now(),
	}
	if err := r.roleRepository.Create(ctx, newRole); err != nil {
		return err
	}
	return nil
}

// DeleteRole implements RoleService.
func (r *RoleServiceDefault) DeleteRole(ctx context.Context, roleName string) error {
	if err := r.roleRepository.Delete(ctx, roleName); err != nil {
		return err
	}
	return nil
}

// RemovePermission implements RoleService.
func (r *RoleServiceDefault) RemovePermission(ctx context.Context, roleName string, permissionKey string) error {
	role, err := r.roleRepository.Read(ctx, roleName)
	if err != nil {
		return err
	}

	role.RemovePermission(permissionKey)
	if err := r.roleRepository.Update(ctx, *role); err != nil {
		return err
	}
	return nil
}

// UpdateRole implements RoleService.
func (r *RoleServiceDefault) UpdateRole(ctx context.Context, roleName string, description string) error {
	role, err := r.roleRepository.Read(ctx, roleName)
	if err != nil {
		return err
	}

	role.Description = description
	if err := r.roleRepository.Update(ctx, *role); err != nil {
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
	CreatePermission(ctx context.Context, name string, action string) error
	DeletePermission(ctx context.Context, name string, action string) error
	ListPermissions(ctx context.Context, paging shared.PageOptions) ([]Permission, error)
	DoesPermissionExist(ctx context.Context, permissionKey string) (bool, error)
}

type PermissionServiceDefault struct {
	permissionRepository PermissionsRepository
}

// CreatePermission implements PermissionService.
func (p PermissionServiceDefault) CreatePermission(ctx context.Context, name string, action string) error {
	newPermission := NewPermission(name, action)
	if err := p.permissionRepository.Create(ctx, newPermission); err != nil {
		return err
	}
	return nil
}

// DeletePermission implements PermissionService.
func (p PermissionServiceDefault) DeletePermission(ctx context.Context, name string, action string) error {
	if err := p.permissionRepository.Delete(ctx, fmt.Sprintf("%s:%s", name, action)); err != nil {
		return err
	}
	return nil
}

// DoesPermissionExist implements PermissionService.
func (p PermissionServiceDefault) DoesPermissionExist(ctx context.Context, permissionKey string) (bool, error) {
	_, err := p.permissionRepository.Read(ctx, permissionKey)
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListPermission implements PermissionService.
func (p PermissionServiceDefault) ListPermissions(ctx context.Context, paging shared.PageOptions) ([]Permission, error) {
	permissions, err := p.permissionRepository.ReadAll(ctx, paging)
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
