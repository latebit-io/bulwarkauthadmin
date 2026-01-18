# Tenant Middleware Usage Guide

## Overview

The `TenantMiddleware` provides a centralized way to extract, validate, and authorize tenant access from URL paths across all handlers.

## How It Works

1. **Middleware extracts** the `:tenantid` parameter from the URL
2. **Validates** the tenant exists in the database
3. **Authorizes** the user has access to the tenant (system admin or tenant member)
4. **Stores** the tenant ID in the request context
5. **Handlers retrieve** the tenant ID using helper functions

## Authorization Logic

- **System Admins** (`bulwark_admin` role): Can access ANY tenant
- **Tenant Users**: Can only access their own tenant (tenantID in JWT must match URL tenantID)

## Two Middleware Options

### 1. `ExtractTenant` - Validation Only
Use this when you want to validate the tenant exists but handle authorization manually in the handler.

### 2. `ExtractAndAuthorizeTenant` - Validation + Authorization (Recommended)
Use this for automatic tenant access control. Blocks requests if user doesn't have access.

## Setup in main.go

```go
import (
    bulwarkauthmiddleware "github.com/latebit-io/bulwarkauthadmin/api/middleware"
    // ... other imports
)

func main() {
    // ... existing setup ...
    
    // JWT middleware must run BEFORE tenant middleware (sets claims)
    jwt := bulwarkauthmiddleware.NewJWTMiddleware(bulwarkGuard)
    service.Use(jwt.Jwt)
    
    // Create tenant middleware
    tenantService := tenants.NewDefaultTenantService(tenantRepository)
    tenantMiddleware := bulwarkauthmiddleware.NewTenantMiddleware(tenantService)
    
    // Apply middleware to tenant-scoped route groups
    tenantGroup := service.Group("/api/v1/tenant/:tenantid")
    tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant) // With authorization
    
    // Register routes on the group (remove /api/v1/tenant/:tenantid prefix from routes)
    accountsHandler := accountsapi.NewAccountHandler(accountsManagmentService)
    accountsapi.AccountRoutesV1(tenantGroup, accountsHandler)
    
    rbacHandler := rbacapi.NewRbacHandler(roleService, permissionService)
    rbacapi.RbacRoutesV1(tenantGroup, rbacHandler)
}
```

## Update Route Definitions

**Before (in account_routes.go):**
```go
func AccountRoutesV1(e *echo.Echo, handler *AccountHandler) {
    e.POST("/api/v1/tenant/:tenantid/accounts", handler.RegisterAccount)
    e.GET("/api/v1/tenant/:tenantid/accounts/:id", handler.GetAccount)
}
```

**After (in account_routes.go):**
```go
func AccountRoutesV1(g *echo.Group, handler *AccountHandler) {
    g.POST("/accounts", handler.RegisterAccount)
    g.GET("/accounts/:id", handler.GetAccount)
    g.GET("/accounts", handler.ListAccounts)
    // ... other routes
}
```

## Using in Handlers

```go
package accounts

import (
    "github.com/labstack/echo/v4"
    bulwarkauthmiddleware "github.com/latebit-io/bulwarkauthadmin/api/middleware"
)

func (ah AccountHandler) RegisterAccount(c echo.Context) error {
    // Extract tenant ID from context (validated by middleware)
    tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)
    
    // Now use tenantID in your business logic
    newAccountRequest := new(NewAccountRequest)
    if err := c.Bind(newAccountRequest); err != nil {
        httpError := problem.NewBadRequest(err)
        return echo.NewHTTPError(httpError.Status, httpError)
    }

    ctx := c.Request().Context()
    
    // Pass tenantID to service layer
    err := ah.accounts.RegisterAccount(ctx, tenantID, newAccountRequest.Email, accounts.AccountOptions{
        IsVerified: true,
    })
    
    if err != nil {
        // ... error handling
    }

    return c.NoContent(http.StatusCreated)
}

func (ah AccountHandler) GetAccount(c echo.Context) error {
    tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)
    accountID := c.Param("id")
    
    ctx := c.Request().Context()
    account, err := ah.accounts.GetAccountDetails(ctx, tenantID, accountID)
    // ... rest of handler
}
```

## Helper Functions Available

### In Handlers

```go
// Get tenant ID from context
tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)

// Get user claims (to check roles manually if needed)
claims, ok := bulwarkauthmiddleware.GetAccountClaims(c)
if ok {
    if bulwarkauthmiddleware.IsSystemAdmin(claims) {
        // User is a system admin
    }
}

// Check if user can access a specific tenant (manual check)
if !bulwarkauthmiddleware.CanAccessTenant(claims, tenantID) {
    return echo.ErrForbidden
}
```

## Benefits

1. **DRY Principle**: Tenant validation and authorization logic written once
2. **Consistent Error Handling**: Standard error responses for invalid/missing tenants and unauthorized access
3. **Type Safety**: Tenant ID stored in context with proper type
4. **Performance**: Tenant validation happens once per request
5. **Security**: 
   - Ensures tenant exists before any business logic executes
   - Automatically enforces tenant isolation
   - System admins can manage all tenants
   - Regular users can only access their assigned tenant
6. **Easy Testing**: Mock the tenant service for middleware tests

## Error Responses

The `ExtractAndAuthorizeTenant` middleware automatically returns:

- **400 Bad Request**: If tenant ID is missing from URL
- **401 Unauthorized**: If user is not authenticated (JWT claims missing)
- **403 Forbidden**: If user doesn't have access to the requested tenant
- **404 Not Found**: If tenant doesn't exist
- **500 Internal Server Error**: If database error occurs during validation

## Example Use Cases

### Scenario 1: System Admin Managing Multiple Tenants
```
User JWT Claims:
  - TenantID: "00000000-0000-0000-0000-000000000000" (System)
  - Roles: ["bulwark_admin"]

Request: GET /api/v1/tenant/tenant-abc-123/accounts
Result: ✅ ALLOWED - System admin can access any tenant
```

### Scenario 2: Tenant Admin Accessing Own Tenant
```
User JWT Claims:
  - TenantID: "tenant-abc-123"
  - Roles: ["tenant_admin"]

Request: GET /api/v1/tenant/tenant-abc-123/accounts
Result: ✅ ALLOWED - TenantID matches
```

### Scenario 3: Tenant Admin Trying to Access Another Tenant
```
User JWT Claims:
  - TenantID: "tenant-abc-123"
  - Roles: ["tenant_admin"]

Request: GET /api/v1/tenant/tenant-xyz-789/accounts
Result: ❌ FORBIDDEN - TenantID doesn't match
```

### Scenario 4: System Admin Accessing System Tenant
```
User JWT Claims:
  - TenantID: "00000000-0000-0000-0000-000000000000"
  - Roles: ["bulwark_admin"]

Request: GET /api/v1/tenant/00000000-0000-0000-0000-000000000000/accounts
Result: ✅ ALLOWED - System admin accessing system tenant
```

## Next Steps

After implementing the middleware, you need to:

1. Update all service interfaces to accept `tenantID string` as first parameter
2. Update all repository methods to filter by tenant ID
3. Add `TenantID` field to all domain models (Account, Role, Permission)
4. Update MongoDB queries to include tenant filter
5. Add database indexes on `{tenantid, ...}` for performance
