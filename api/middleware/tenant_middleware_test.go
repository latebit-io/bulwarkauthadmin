package middleware

import (
	"testing"
)

func TestIsTenantAdmin(t *testing.T) {
	tests := []struct {
		name     string
		claims   AccountClaims
		expected bool
	}{
		{
			name: "user with tenant_admin role",
			claims: AccountClaims{
				Roles: []string{"tenant_admin"},
			},
			expected: true,
		},
		{
			name: "user with tenant_admin and other roles",
			claims: AccountClaims{
				Roles: []string{"user", "tenant_admin", "viewer"},
			},
			expected: true,
		},
		{
			name: "user without tenant_admin role",
			claims: AccountClaims{
				Roles: []string{"user", "viewer"},
			},
			expected: false,
		},
		{
			name: "user with no roles",
			claims: AccountClaims{
				Roles: []string{},
			},
			expected: false,
		},
		{
			name: "user with nil roles",
			claims: AccountClaims{
				Roles: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTenantAdmin(tt.claims)
			if result != tt.expected {
				t.Errorf("IsTenantAdmin() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsSystemAdmin(t *testing.T) {
	tests := []struct {
		name     string
		claims   AccountClaims
		expected bool
	}{
		{
			name: "user with bulwark_admin role",
			claims: AccountClaims{
				Roles: []string{"bulwark_admin"},
			},
			expected: true,
		},
		{
			name: "user with bulwark_admin and other roles",
			claims: AccountClaims{
				Roles: []string{"user", "bulwark_admin"},
			},
			expected: true,
		},
		{
			name: "user without bulwark_admin role",
			claims: AccountClaims{
				Roles: []string{"tenant_admin", "user"},
			},
			expected: false,
		},
		{
			name: "user with no roles",
			claims: AccountClaims{
				Roles: []string{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSystemAdmin(tt.claims)
			if result != tt.expected {
				t.Errorf("IsSystemAdmin() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsTenantAdminOrSystemAdmin(t *testing.T) {
	tests := []struct {
		name     string
		claims   AccountClaims
		expected bool
	}{
		{
			name: "user with tenant_admin role",
			claims: AccountClaims{
				Roles: []string{"tenant_admin"},
			},
			expected: true,
		},
		{
			name: "user with bulwark_admin role",
			claims: AccountClaims{
				Roles: []string{"bulwark_admin"},
			},
			expected: true,
		},
		{
			name: "user with both roles",
			claims: AccountClaims{
				Roles: []string{"tenant_admin", "bulwark_admin"},
			},
			expected: true,
		},
		{
			name: "user with neither role",
			claims: AccountClaims{
				Roles: []string{"user", "viewer"},
			},
			expected: false,
		},
		{
			name: "user with no roles",
			claims: AccountClaims{
				Roles: []string{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTenantAdminOrSystemAdmin(tt.claims)
			if result != tt.expected {
				t.Errorf("IsTenantAdminOrSystemAdmin() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCanAccessTenant(t *testing.T) {
	systemAdminClaims := AccountClaims{
		Roles:    []string{"bulwark_admin"},
		TenantID: "system-tenant",
	}

	userTenantClaims := AccountClaims{
		Roles:    []string{"user"},
		TenantID: "tenant-123",
	}

	tests := []struct {
		name              string
		claims            AccountClaims
		requestedTenantID string
		expected          bool
	}{
		{
			name:              "system admin can access any tenant",
			claims:            systemAdminClaims,
			requestedTenantID: "tenant-456",
			expected:          true,
		},
		{
			name:              "user can access their own tenant",
			claims:            userTenantClaims,
			requestedTenantID: "tenant-123",
			expected:          true,
		},
		{
			name:              "user cannot access different tenant",
			claims:            userTenantClaims,
			requestedTenantID: "tenant-456",
			expected:          false,
		},
		{
			name: "system admin can access system tenant",
			claims: AccountClaims{
				Roles:    []string{"bulwark_admin"},
				TenantID: "00000000-0000-0000-0000-000000000000",
			},
			requestedTenantID: "00000000-0000-0000-0000-000000000000",
			expected:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanAccessTenant(tt.claims, tt.requestedTenantID)
			if result != tt.expected {
				t.Errorf("CanAccessTenant() = %v, want %v", result, tt.expected)
			}
		})
	}
}
