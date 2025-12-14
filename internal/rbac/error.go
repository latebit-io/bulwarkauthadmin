package rbac

import "fmt"

type RoleNotFoundError struct {
	Value string `json:"value"`
}

func (e RoleNotFoundError) Error() string {
	return fmt.Sprintf("role not found: '%s'", e.Value)
}

type RoleDuplicateError struct {
	Value string `json:"value"`
}

func (e RoleDuplicateError) Error() string {
	return fmt.Sprintf("duplicate role: '%s' already exists", e.Value)
}

type PermissionNotFoundError struct {
	Value string `json:"value"`
}

func (e PermissionNotFoundError) Error() string {
	return fmt.Sprintf("permission not found: '%s'", e.Value)
}

type PermissionDuplicateError struct {
	Value string `json:"value"`
}

func (e PermissionDuplicateError) Error() string {
	return fmt.Sprintf("duplicate role: '%s' already exists", e.Value)
}
