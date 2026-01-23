package middleware

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
	"github.com/latebit-io/bulwarkauthadmin/internal/tenants"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenantID"

	// System admin role name
	SystemAdminRole = "bulwark_admin"

	// Tenant admin role name
	TenantAdminRole = "tenant_admin"

	// System tenant ID (UUID nil)
	SystemTenantID = "00000000-0000-0000-0000-000000000000"
)

// TenantMiddleware extracts and validates tenant ID from URL path parameter
type TenantMiddleware struct {
	tenantService tenants.TenantService
}

// NewTenantMiddleware creates a new tenant middleware instance
func NewTenantMiddleware(tenantService tenants.TenantService) *TenantMiddleware {
	return &TenantMiddleware{
		tenantService: tenantService,
	}
}

// IsSystemAdmin checks if the user has the system admin role
func IsSystemAdmin(claims AccountClaims) bool {
	for _, role := range claims.Roles {
		if role == SystemAdminRole {
			return true
		}
	}
	return false
}

// IsTenantAdmin checks if the user has the tenant admin role
func IsTenantAdmin(claims AccountClaims) bool {
	for _, role := range claims.Roles {
		if role == TenantAdminRole {
			return true
		}
	}
	return false
}

// IsTenantAdminOrSystemAdmin checks if the user has either tenant admin or system admin role
// System admins implicitly have tenant admin permissions
func IsTenantAdminOrSystemAdmin(claims AccountClaims) bool {
	return IsTenantAdmin(claims) || IsSystemAdmin(claims)
}

// CanAccessTenant checks if a user can access a specific tenant
// Returns true if:
// - User is a system admin (can access any tenant)
// - User's tenant ID in JWT matches the requested tenant ID
func CanAccessTenant(claims AccountClaims, requestedTenantID string) bool {
	// System admins can access any tenant
	if IsSystemAdmin(claims) {
		return true
	}

	// Users can only access their own tenant
	return claims.TenantID == requestedTenantID
}

// GetAccountClaims retrieves the account claims from echo context
// These claims are set by the JWT middleware
func GetAccountClaims(c echo.Context) (AccountClaims, bool) {
	claims, ok := c.Get("claims").(AccountClaims)
	return claims, ok
}

// ExtractTenant is middleware that extracts tenant ID from URL and validates it exists
func (tm *TenantMiddleware) ExtractTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantID := c.Param("tenantid")

		// Validate tenant ID is present
		if tenantID == "" {
			httpError := problem.NewBadRequest(nil)
			httpError.Detail = "tenant ID is required in URL path"
			return echo.NewHTTPError(httpError.Status, httpError)
		}

		// Validate tenant exists
		ctx := c.Request().Context()
		_, err := tm.tenantService.GetTenant(ctx, tenantID)
		if err != nil {
			var tenantNotFoundError tenants.TenantNotFoundError
			if err == tenantNotFoundError || err.Error() == "tenant not found" {
				return echo.NewHTTPError(http.StatusNotFound, problem.Details{
					Type:   "https://latebit.io/bulwark/errors/tenant-not-found",
					Title:  "Tenant Not Found",
					Status: http.StatusNotFound,
					Detail: "The specified tenant does not exist",
				})
			}

			httpError := problem.NewServerError(err)
			return echo.NewHTTPError(httpError.Status, httpError)
		}

		// Add tenant ID to request context
		ctx = context.WithValue(ctx, tenantIDKey, tenantID)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// ExtractAndAuthorizeTenant is middleware that extracts tenant ID, validates it exists,
// and checks if the authenticated user has access to this tenant
func (tm *TenantMiddleware) ExtractAndAuthorizeTenant(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tenantID := c.Param("tenantid")

		// Validate tenant ID is present
		if tenantID == "" {
			httpError := problem.NewBadRequest(nil)
			httpError.Detail = "tenant ID is required in URL path"
			return echo.NewHTTPError(httpError.Status, httpError)
		}

		// Validate tenant exists
		ctx := c.Request().Context()
		_, err := tm.tenantService.GetTenant(ctx, tenantID)
		if err != nil {
			var tenantNotFoundError tenants.TenantNotFoundError
			if err == tenantNotFoundError || err.Error() == "tenant not found" {
				return echo.NewHTTPError(http.StatusNotFound, problem.Details{
					Type:   "https://latebit.io/bulwark/errors/tenant-not-found",
					Title:  "Tenant Not Found",
					Status: http.StatusNotFound,
					Detail: "The specified tenant does not exist",
				})
			}

			httpError := problem.NewServerError(err)
			return echo.NewHTTPError(httpError.Status, httpError)
		}

		// Get authenticated user's claims
		claims, ok := GetAccountClaims(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/unauthorized",
				Title:  "Unauthorized",
				Status: http.StatusUnauthorized,
				Detail: "Authentication required",
			})
		}

		// Check if user can access this tenant
		if !CanAccessTenant(claims, tenantID) {
			return echo.NewHTTPError(http.StatusForbidden, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/forbidden",
				Title:  "Access Denied",
				Status: http.StatusForbidden,
				Detail: "You do not have access to this tenant",
			})
		}

		// Add tenant ID to request context
		ctx = context.WithValue(ctx, tenantIDKey, tenantID)
		c.SetRequest(c.Request().WithContext(ctx))

		return next(c)
	}
}

// GetTenantID retrieves the tenant ID from the request context
// This should be called from handlers after the TenantMiddleware has run
func GetTenantID(ctx context.Context) string {
	tenantID, ok := ctx.Value(tenantIDKey).(string)
	if !ok {
		return ""
	}
	return tenantID
}

// GetTenantIDFromEcho is a convenience function to get tenant ID from echo.Context
func GetTenantIDFromEcho(c echo.Context) string {
	return GetTenantID(c.Request().Context())
}

// RequireTenantAdminOrSystemAdmin is middleware that ensures the authenticated user has tenant admin or system admin role
// This middleware must run AFTER JWT middleware and ExtractAndAuthorizeTenant (which set the claims)
func (tm *TenantMiddleware) RequireTenantAdminOrSystemAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get claims from context (set by JWT middleware)
		claims, ok := GetAccountClaims(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/unauthorized",
				Title:  "Unauthorized",
				Status: http.StatusUnauthorized,
				Detail: "Authentication required",
			})
		}

		// Check if user is tenant admin or system admin
		if !IsTenantAdminOrSystemAdmin(claims) {
			return echo.NewHTTPError(http.StatusForbidden, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/forbidden",
				Title:  "Access Denied",
				Status: http.StatusForbidden,
				Detail: "Tenant administrator access required",
			})
		}

		return next(c)
	}
}
