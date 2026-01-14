package tenants

import "fmt"

type TenantNotFoundError struct {
	Value string `json:"value"`
}

func (e TenantNotFoundError) Error() string {
	return fmt.Sprintf("tenant not found: '%s'", e.Value)
}

type TenantDuplicateError struct {
	Value string `json:"value"`
}

func (e TenantDuplicateError) Error() string {
	return fmt.Sprintf("duplicate tenant: '%s' already exists", e.Value)
}
