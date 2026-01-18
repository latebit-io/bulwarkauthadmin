# System Admin Routes - Implementation Guide

## Overview

Tenant management routes (`/api/v1/admin/tenants/*`) are now restricted to **system administrators only** using a combination of:

1. **JWT Middleware** - Validates token against system tenant
2. **RequireSystemAdmin Middleware** - Ensures user has `bulwark_admin` role

## Files Created/Modified

### New Files

1. **`api/middleware/system_admin_middleware.go`** - Enforces system admin role
2. **`api/middleware/jwt.go`** - Added `JwtForSystemRoutes()` method

### Modified Files

1. **`api/tenants/tenant_routes.go`** - Changed to use `*echo.Group`, added full CRUD routes
2. **`api/tenants/tenant_handlers.go`** - Added ListTenants, GetTenant, UpdateTenant, DeleteTenant handlers

## How to Setup in main.go

```go
func main() {
    logger := getLogger()
    // ... banner, config, mongodb setup ...
    
    service := echo.New()
    service.HideBanner = true
    
    httpClient := &http.Client{}
    bulwarkGuard := bulwark.NewGuard(config.BulwarkAuthUrl, httpClient)
    
    // Create JWT middleware
    jwt := bulwarkauthmiddleware.NewJWTMiddleware(bulwarkGuard)
    
    mongodb := client.Database("bulwarkauth" + config.DbNameSeed)
    
    // ==========================================
    // SYSTEM ADMIN ROUTES (No tenantID in URL)
    // ==========================================
    adminGroup := service.Group("/api/v1/admin")
    adminGroup.Use(jwt.JwtForSystemRoutes)                      // 1. Validate JWT with system tenant
    adminGroup.Use(bulwarkauthmiddleware.RequireSystemAdmin)    // 2. Require bulwark_admin role
    
    // Tenant management (system admins only)
    tenantRepository := tenants.NewMongoDbTenantRepository(mongodb)
    err = tenantRepository.CreateSystem(context.Background())
    if err != nil {
        panic(err)
    }
    tenantService := tenants.NewDefaultTenantService(tenantRepository)
    tenantsHandler := tenantsapi.NewTenantHandler(tenantService)
    tenantsapi.TenantRoutesV1(adminGroup, tenantsHandler)
    
    // ==========================================
    // TENANT-SCOPED ROUTES (With :tenantid in URL)
    // ==========================================
    tenantMiddleware := bulwarkauthmiddleware.NewTenantMiddleware(tenantService)
    
    tenantGroup := service.Group("/api/v1/tenant/:tenantid")
    tenantGroup.Use(jwt.Jwt)                                    // 1. Validate JWT with URL tenantID
    tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant) // 2. Authorize tenant access
    
    // Account routes
    accountRepository := accounts.NewMongoDBAccountRepository(mongodb)
    accountsManagmentService := accounts.NewAccountManagementServiceDefault(accountRepository)
    accountsHandler := accountsapi.NewAccountHandler(accountsManagmentService)
    accountsapi.AccountRoutesV1(tenantGroup, accountsHandler)
    
    // RBAC routes
    permissionsRepository := rbac.NewMongoDBPermissionsRepository(mongodb)
    rolesRepository := rbac.NewMongoDBRolesRepository(mongodb)
    roleService := rbac.NewRoleServiceDefault(rolesRepository)
    permissionService := rbac.NewPermissionServiceDefault(permissionsRepository)
    rbacHandler := rbacapi.NewRbacHandler(roleService, permissionService)
    rbacapi.RbacRoutesV1(tenantGroup, rbacHandler)
    
    // Account RBAC routes
    accountsRBAC := accountsRbac.NewAccountRBACServiceDefault(accountRepository, permissionService, roleService)
    accountsRbacHandler := accountsrbacapi.NewAccountRBACHandler(accountsRBAC)
    accountsrbacapi.AccountRBACRoutesV1(tenantGroup, accountsRbacHandler)
    
    // ==========================================
    // ADMIN ACCOUNT SETUP
    // ==========================================
    adminAccountService := adminAccount.NewAdminAccountsServiceDefault(
        accountRepository,
        rolesRepository,
        permissionsRepository,
        accountsRBAC,
        bulwarkGuard,
    )
    
    err = adminAccountService.CreateInternalRoles(context.Background())
    if err != nil {
        logger.Error("failed to create internal roles", "error", err)
        panic(err)
    }
    
    defaultAdminAccount := config.AdminAccount
    defaultAdminPassword := config.AdminAccountPassword
    if defaultAdminAccount != "" {
        err = adminAccountService.RegisterAccount(context.Background(), defaultAdminAccount, defaultAdminPassword)
        if err != nil {
            logger.Error("could configure default admin account", "error", err)
            panic(err)
        }
    }
    
    // ==========================================
    // PUBLIC ROUTES (No authentication)
    // ==========================================
    healthHandler := health.NewHealthHandler()
    health.HealthRoutes(service, healthHandler)
    
    // CORS must be configured
    corsSetting(service, config, logger)
    
    // Start server
    if err := service.Start(fmt.Sprintf(":%d", config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.Error(err.Error())
    }
}
```

## Route Structure

### System Admin Routes (No tenant in URL)
```
POST   /api/v1/admin/tenants           - Create new tenant
GET    /api/v1/admin/tenants           - List all tenants
GET    /api/v1/admin/tenants/:id       - Get tenant details
PUT    /api/v1/admin/tenants/:id       - Update tenant
DELETE /api/v1/admin/tenants/:id       - Delete tenant
```

**Access:** System administrators only (`bulwark_admin` role)

### Tenant-Scoped Routes (With tenant in URL)
```
POST   /api/v1/tenant/:tenantid/accounts                      - Register account
GET    /api/v1/tenant/:tenantid/accounts                      - List accounts
GET    /api/v1/tenant/:tenantid/accounts/:id                  - Get account
PUT    /api/v1/tenant/:tenantid/accounts/email                - Change email
PUT    /api/v1/tenant/:tenantid/accounts/disable              - Disable account
PUT    /api/v1/tenant/:tenantid/accounts/enable               - Enable account
PUT    /api/v1/tenant/:tenantid/accounts/deactivate           - Deactivate account
PUT    /api/v1/tenant/:tenantid/accounts/unlink               - Unlink social provider

POST   /api/v1/tenant/:tenantid/roles                         - Create role
GET    /api/v1/tenant/:tenantid/roles                         - List roles
GET    /api/v1/tenant/:tenantid/roles/:id                     - Get role
PUT    /api/v1/tenant/:tenantid/roles/:id                     - Update role
DELETE /api/v1/tenant/:tenantid/roles/:id                     - Delete role
PUT    /api/v1/tenant/:tenantid/roles/:id/permissions         - Add permission to role
DELETE /api/v1/tenant/:tenantid/roles/:id/permissions/:permissionId - Remove permission

POST   /api/v1/tenant/:tenantid/permissions                   - Create permission
GET    /api/v1/tenant/:tenantid/permissions                   - List permissions
GET    /api/v1/tenant/:tenantid/permissions/exists/:id        - Check permission exists
DELETE /api/v1/tenant/:tenantid/permissions/:id               - Delete permission
```

**Access:** 
- System admins can access ANY tenant
- Regular users can only access THEIR OWN tenant

### Public Routes (No authentication)
```
GET    /health                         - Health check
```

## Authorization Flow

### System Admin Routes Flow

```
HTTP Request: POST /api/v1/admin/tenants
Authorization: Bearer <jwt-token>
Body: {"name": "Acme Corp", ...}
         ↓
┌───────────────────────────────────────────┐
│ 1. JwtForSystemRoutes Middleware         │
│    - Validates JWT with system tenant ID  │
│      (00000000-0000-0000-0000-000000000000)│
│    - Stores claims in context             │
└───────────────────────────────────────────┘
         ↓
┌───────────────────────────────────────────┐
│ 2. RequireSystemAdmin Middleware         │
│    - Gets claims from context             │
│    - Checks if "bulwark_admin" in roles   │
│    - ALLOW if yes, 403 if no              │
└───────────────────────────────────────────┘
         ↓
┌───────────────────────────────────────────┐
│ 3. Handler (AddTenant)                    │
│    - Creates tenant in database           │
│    - Returns 201 Created                  │
└───────────────────────────────────────────┘
```

### Tenant-Scoped Routes Flow

```
HTTP Request: GET /api/v1/tenant/tenant-123/accounts
Authorization: Bearer <jwt-token>
         ↓
┌───────────────────────────────────────────┐
│ 1. JWT Middleware                         │
│    - Extracts tenantID from URL: "tenant-123"│
│    - Validates JWT with that tenant ID    │
│    - Stores claims in context             │
└───────────────────────────────────────────┘
         ↓
┌───────────────────────────────────────────┐
│ 2. ExtractAndAuthorizeTenant Middleware  │
│    - Validates tenant exists              │
│    - Checks authorization:                │
│      • Is system admin? → ALLOW           │
│      • JWT tenantID == URL tenantID?      │
│        → ALLOW, else 403                  │
└───────────────────────────────────────────┘
         ↓
┌───────────────────────────────────────────┐
│ 3. Handler (ListAccounts)                 │
│    - Gets tenantID from context           │
│    - Lists accounts for that tenant       │
│    - Returns 200 OK                       │
└───────────────────────────────────────────┘
```

## Example API Calls

### 1. System Admin Creates a Tenant

```bash
POST /api/v1/admin/tenants HTTP/1.1
Authorization: Bearer <system-admin-jwt>
Content-Type: application/json

{
  "name": "Acme Corporation",
  "description": "Main tenant for Acme Corp",
  "domain": "acme.com"
}

# Response: 201 Created
```

**Requirements:**
- JWT must be from system tenant (`00000000-0000-0000-0000-000000000000`)
- User must have `bulwark_admin` role

### 2. System Admin Lists All Tenants

```bash
GET /api/v1/admin/tenants HTTP/1.1
Authorization: Bearer <system-admin-jwt>

# Response: 200 OK
[
  {
    "id": "00000000-0000-0000-0000-000000000000",
    "name": "System",
    "description": "System tenant",
    "created_at": "2024-01-01T00:00:00Z"
  },
  {
    "id": "tenant-abc-123",
    "name": "Acme Corporation",
    "description": "Main tenant for Acme Corp",
    "domain": "acme.com",
    "created_at": "2024-01-15T10:30:00Z"
  }
]
```

### 3. System Admin Manages Any Tenant's Accounts

```bash
GET /api/v1/tenant/tenant-abc-123/accounts HTTP/1.1
Authorization: Bearer <system-admin-jwt>

# Response: 200 OK - System admin can access any tenant
```

### 4. Regular User Tries to Create Tenant (Fails)

```bash
POST /api/v1/admin/tenants HTTP/1.1
Authorization: Bearer <tenant-user-jwt>
Content-Type: application/json

{
  "name": "Evil Corp",
  "description": "Should not be allowed"
}

# Response: 403 Forbidden
{
  "type": "https://latebit.io/bulwark/errors/forbidden",
  "title": "Access Denied",
  "status": 403,
  "detail": "System administrator access required"
}
```

### 5. Regular User Accesses Own Tenant (Success)

```bash
GET /api/v1/tenant/tenant-abc-123/accounts HTTP/1.1
Authorization: Bearer <tenant-abc-123-user-jwt>

# Response: 200 OK - User can access their own tenant
```

### 6. Regular User Tries to Access Another Tenant (Fails)

```bash
GET /api/v1/tenant/tenant-xyz-789/accounts HTTP/1.1
Authorization: Bearer <tenant-abc-123-user-jwt>

# Response: 403 Forbidden
{
  "type": "https://latebit.io/bulwark/errors/forbidden",
  "title": "Access Denied",
  "status": 403,
  "detail": "You do not have access to this tenant"
}
```

## Middleware Summary

| Middleware | Purpose | When to Use | Applied To |
|-----------|---------|-------------|-----------|
| `jwt.Jwt()` | Validates JWT with tenant ID from URL | Routes with `:tenantid` parameter | Tenant-scoped routes |
| `jwt.JwtForSystemRoutes()` | Validates JWT with system tenant ID | Routes without tenant parameter | Admin routes |
| `tenantMiddleware.ExtractAndAuthorizeTenant()` | Validates tenant + checks access | After JWT middleware | Tenant-scoped routes |
| `RequireSystemAdmin()` | Enforces `bulwark_admin` role | Admin-only operations | Admin routes |

## Security Guarantees

✅ **Tenant management is restricted to system admins only**
- Regular users get 403 Forbidden when accessing `/api/v1/admin/tenants/*`
- Only users with `bulwark_admin` role can create/update/delete tenants

✅ **Tenant isolation is enforced**
- Users can only access their own tenant's data
- System admins can access all tenants (for support/management)

✅ **JWT validation includes tenant context**
- Tokens are validated against the appropriate tenant
- Prevents token reuse across tenants

✅ **No authentication bypasses**
- All protected routes require valid JWT
- All admin routes require system admin role
- Middleware order ensures security

## Testing Checklist

- [ ] System admin can create tenants
- [ ] System admin can list all tenants
- [ ] System admin can update any tenant
- [ ] System admin can delete tenants
- [ ] System admin can access any tenant's resources
- [ ] Regular user gets 403 when accessing `/api/v1/admin/tenants`
- [ ] Regular user can access their own tenant
- [ ] Regular user gets 403 when accessing other tenants
- [ ] Unauthenticated requests get 401
- [ ] Invalid tenant IDs return 404

## Migration Notes

If you have existing tenants in the database:
1. Ensure system tenant exists (UUID nil: `00000000-0000-0000-0000-000000000000`)
2. Verify system admin account exists with `bulwark_admin` role
3. Test system admin can access `/api/v1/admin/tenants` endpoint
4. Test regular users are blocked from tenant management endpoints
