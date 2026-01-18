# Tenant Authorization Solution - Complete Implementation

## Summary

Your decision to include tenant ID in the URL is **100% correct** from a staff engineer perspective. I've built a complete authorization solution that ensures:

1. ✅ **System admins** (`bulwark_admin` role) can access ALL tenants
2. ✅ **Regular users** can ONLY access their assigned tenant
3. ✅ **Automatic enforcement** via middleware (no repetitive code in handlers)
4. ✅ **Secure by default** - requests fail if tenant access is not allowed

## What Was Built

### 1. Tenant Middleware (`api/middleware/tenant_middleware.go`)

Two middleware functions:

#### `ExtractAndAuthorizeTenant` (Recommended - Use This)
- Validates tenant exists in database
- Checks JWT claims to authorize access
- System admins bypass restrictions
- Regular users must match JWT tenantID with URL tenantID
- Returns 403 Forbidden if unauthorized

#### `ExtractTenant` (Validation Only)
- Only validates tenant exists
- Use when you need custom authorization logic

### 2. Helper Functions

```go
// Get validated tenant ID in any handler
tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)

// Get user claims
claims, ok := bulwarkauthmiddleware.GetAccountClaims(c)

// Check if user is system admin
if bulwarkauthmiddleware.IsSystemAdmin(claims) {
    // Has bulwark_admin role
}

// Check if user can access a tenant
if bulwarkauthmiddleware.CanAccessTenant(claims, requestedTenantID) {
    // Allowed
}
```

### 3. Constants

```go
const (
    SystemAdminRole = "bulwark_admin"
    SystemTenantID  = "00000000-0000-0000-0000-000000000000"
)
```

## How to Integrate into main.go

```go
func main() {
    // ... existing MongoDB and service setup ...
    
    // STEP 1: JWT middleware must run FIRST (sets claims)
    jwt := bulwarkauthmiddleware.NewJWTMiddleware(bulwarkGuard)
    service.Use(jwt.Jwt)
    
    // STEP 2: Create tenant middleware
    tenantService := tenants.NewDefaultTenantService(tenantRepository)
    tenantMiddleware := bulwarkauthmiddleware.NewTenantMiddleware(tenantService)
    
    // STEP 3: Create tenant route group with authorization
    tenantGroup := service.Group("/api/v1/tenant/:tenantid")
    tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant)
    
    // STEP 4: Register routes on the group (without /api/v1/tenant/:tenantid prefix)
    accountsHandler := accountsapi.NewAccountHandler(accountsManagmentService)
    accountsapi.AccountRoutesV1(tenantGroup, accountsHandler)
    
    rbacHandler := rbacapi.NewRbacHandler(roleService, permissionService)
    rbacapi.RbacRoutesV1(tenantGroup, rbacHandler)
    
    accountsRbacHandler := accountsrbacapi.NewAccountRBACHandler(accountsRBAC)
    accountsrbacapi.AccountRBACRoutesV1(tenantGroup, accountsRbacHandler)
    
    // Health check doesn't need tenant middleware
    healthHandler := health.NewHealthHandler()
    health.HealthRoutes(service, healthHandler)
    
    // ... rest of main ...
}
```

## Update Route Definitions

Change route registration functions to accept `*echo.Group` instead of `*echo.Echo`:

**Before:**
```go
func AccountRoutesV1(e *echo.Echo, handler *AccountHandler) {
    e.POST("/api/v1/tenant/:tenantid/accounts", handler.RegisterAccount)
    e.GET("/api/v1/tenant/:tenantid/accounts/:id", handler.GetAccount)
}
```

**After:**
```go
func AccountRoutesV1(g *echo.Group, handler *AccountHandler) {
    g.POST("/accounts", handler.RegisterAccount)
    g.GET("/accounts/:id", handler.GetAccount)
    g.GET("/accounts", handler.ListAccounts)
    // ... other routes
}
```

## Update Handlers to Use Tenant ID

Every handler should extract the tenant ID and pass it to services:

```go
func (ah AccountHandler) RegisterAccount(c echo.Context) error {
    // Extract validated tenant ID (already authorized by middleware)
    tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)
    
    newAccountRequest := new(NewAccountRequest)
    if err := c.Bind(newAccountRequest); err != nil {
        httpError := problem.NewBadRequest(err)
        return echo.NewHTTPError(httpError.Status, httpError)
    }

    ctx := c.Request().Context()
    
    // Pass tenantID to service
    err := ah.accounts.RegisterAccount(ctx, tenantID, newAccountRequest.Email, accounts.AccountOptions{
        IsVerified: true,
    })
    
    if err != nil {
        // ... error handling
    }

    return c.NoContent(http.StatusCreated)
}
```

## Authorization Flow Diagram

```
HTTP Request: GET /api/v1/tenant/tenant-123/accounts
              Authorization: Bearer <jwt>
                        ↓
┌─────────────────────────────────────────────────┐
│ 1. JWT Middleware                               │
│    - Validates JWT token                        │
│    - Extracts claims (roles, tenantID)          │
│    - Stores claims in context                   │
└─────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────┐
│ 2. Tenant Middleware (ExtractAndAuthorizeTenant)│
│    - Extracts tenantID from URL: "tenant-123"   │
│    - Validates tenant exists in DB              │
│    - Gets claims from context                   │
│    - Authorization Check:                       │
│      • Is user system admin? → ALLOW           │
│      • Does JWT tenantID == URL tenantID?      │
│        → ALLOW, else DENY (403)                │
│    - Stores tenantID in context                 │
└─────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────┐
│ 3. Handler                                      │
│    - GetTenantIDFromEcho(c) → "tenant-123"     │
│    - Pass to service layer                      │
└─────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────┐
│ 4. Service Layer                                │
│    - Use tenantID in business logic             │
│    - Pass to repository                         │
└─────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────┐
│ 5. Repository Layer                             │
│    - Filter: {tenantId: "tenant-123", ...}     │
│    - Returns only tenant's data                 │
└─────────────────────────────────────────────────┘
```

## Security Model

### System Admin (bulwark_admin role)
```
JWT Claims:
  - tenantId: "00000000-0000-0000-0000-000000000000"
  - roles: ["bulwark_admin"]

Can access:
  ✅ GET /api/v1/tenant/00000000-0000-0000-0000-000000000000/accounts (system tenant)
  ✅ GET /api/v1/tenant/tenant-abc-123/accounts (any tenant)
  ✅ GET /api/v1/tenant/tenant-xyz-789/accounts (any tenant)
```

### Regular Tenant User
```
JWT Claims:
  - tenantId: "tenant-abc-123"
  - roles: ["tenant_admin"]

Can access:
  ✅ GET /api/v1/tenant/tenant-abc-123/accounts (own tenant)
  ❌ GET /api/v1/tenant/tenant-xyz-789/accounts (403 Forbidden)
  ❌ GET /api/v1/tenant/00000000-0000-0000-0000-000000000000/accounts (403 Forbidden)
```

## What You Already Have ✅

Good news! You've already completed much of the work:

1. ✅ **TenantID field in Account model** - Already added
2. ✅ **Service interfaces accept tenantID** - Already updated
3. ✅ **Repository methods filter by tenantID** - Already implemented
4. ✅ **JWT middleware extracts claims** - Already working
5. ✅ **System admin role defined** - `bulwark_admin` exists
6. ✅ **Tenant service exists** - Can validate tenants

## What's Left to Do

1. **Integrate middleware into main.go** (see code above)
2. **Update route registration** to use `*echo.Group` instead of `*echo.Echo`
3. **Update all handlers** to extract and use `tenantID` from context
4. **Test the authorization** with different user scenarios

## Why This Approach is Correct

From a staff engineer perspective:

### ✅ Pros of Tenant ID in URL
1. **Explicit and auditable** - Every log shows which tenant
2. **REST principle** - Resources are scoped to tenants
3. **Flexible for admin UIs** - Platform admins can switch tenants easily
4. **Clear authorization boundary** - Easy to enforce and test
5. **No ambiguity** - URL states intent clearly

### ❌ Alternative (JWT only) Would Be Wrong For You
- Can't support cross-tenant admin operations
- Makes it harder to audit which tenant was accessed
- Less flexible for platform management
- Doesn't work well with "system" tenant concept

## Testing Checklist

- [ ] System admin can list accounts in any tenant
- [ ] Regular user can list accounts in their own tenant
- [ ] Regular user gets 403 when accessing another tenant
- [ ] Invalid tenant ID returns 404
- [ ] Missing JWT returns 401
- [ ] All database queries filter by tenant ID
- [ ] Cross-tenant data leakage is impossible

## Production Considerations

1. **Logging**: Log both JWT tenantID and URL tenantID for audit trails
2. **Metrics**: Track authorization failures (might indicate attacks)
3. **Rate Limiting**: Consider per-tenant rate limits
4. **Database Indexes**: Ensure all queries have `{tenantId: 1, ...}` indexes
5. **Migration**: Populate tenantID for existing data before enforcing

## Questions to Consider

1. **Can system admins be members of specific tenants?** Currently, system admins bypass all restrictions
2. **Should there be tenant-level roles?** (e.g., tenant_owner, tenant_admin, tenant_user)
3. **Cross-tenant operations?** Do you need any endpoints where users can see data from multiple tenants?
4. **Tenant switching in UI?** How will the frontend handle multi-tenant access for system admins?

## Files Created

1. `api/middleware/tenant_middleware.go` - Core middleware implementation
2. `api/middleware/tenant_middleware_example.md` - Detailed usage guide
3. `api/middleware/TENANT_AUTHORIZATION.md` - Quick reference card
4. This file - Complete solution overview

## Next Steps

Would you like me to:
1. Update your main.go with the middleware integration?
2. Update your route registration files to use echo.Group?
3. Create unit tests for the tenant authorization middleware?
4. Add integration tests that verify tenant isolation?
