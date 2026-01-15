# Corrected Middleware Setup for BulwarkAuthAdmin

## Critical Discovery

Your JWT middleware is **correct as written**, but it's being applied at the **wrong level** in main.go.

### The Problem

```go
// CURRENT (BROKEN) - from main.go line 109
service.Use(jwt.Jwt)  // ❌ Applied globally - c.Param("tenantid") returns ""
```

When JWT middleware runs globally via `service.Use()`, route matching hasn't happened yet, so `c.Param("tenantid")` returns an empty string.

### Why TenantID Is Required

Your `bulwark-auth-guard` library requires the tenant ID because:

1. `ValidateAccessToken(ctx, tenantID, jwt)` sends an HTTP POST to bulwark-auth service:
   ```json
   {
       "tenantId": "tenant-123",
       "token": "eyJhbGc..."
   }
   ```

2. The bulwark-auth service validates:
   - JWT signature is valid
   - JWT was issued for the specified tenant
   - JWT hasn't expired
   - Tenant exists and is active

3. This prevents token reuse across different tenants (multi-tenant security)

## The Solution: Route-Level Middleware

Apply JWT middleware **after** route matching so `c.Param("tenantid")` is available:

### Updated main.go

```go
func main() {
    logger := getLogger()
    // ... banner, config, mongodb setup ...
    
    service := echo.New()
    service.HideBanner = true
    
    // Setup services
    httpClient := &http.Client{}
    bulwarkGuard := bulwark.NewGuard(config.BulwarkAuthUrl, httpClient)
    
    // Create middleware instances
    jwt := bulwarkauthmiddleware.NewJWTMiddleware(bulwarkGuard)
    
    mongodb := client.Database("bulwarkauth" + config.DbNameSeed)
    
    // Tenant setup
    tenantRepository := tenants.NewMongoDbTenantRepository(mongodb)
    err = tenantRepository.CreateSystem(context.Background())
    if err != nil {
        panic(err)
    }
    tenantService := tenants.NewDefaultTenantService(tenantRepository)
    tenantMiddleware := bulwarkauthmiddleware.NewTenantMiddleware(tenantService)
    
    // ✅ CORRECT: Create tenant-scoped route group with BOTH middlewares
    tenantGroup := service.Group("/api/v1/tenant/:tenantid")
    tenantGroup.Use(jwt.Jwt)                                    // 1. JWT auth (needs :tenantid)
    tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant) // 2. Tenant authorization
    
    // Setup account services
    accountRepository := accounts.NewMongoDBAccountRepository(mongodb)
    accountsManagmentService := accounts.NewAccountManagementServiceDefault(accountRepository)
    accountsHandler := accountsapi.NewAccountHandler(accountsManagmentService)
    
    // Register routes WITHOUT /api/v1/tenant/:tenantid prefix
    accountsapi.AccountRoutesV1(tenantGroup, accountsHandler)
    
    // Setup RBAC services
    permissionsRepository := rbac.NewMongoDBPermissionsRepository(mongodb)
    rolesRepository := rbac.NewMongoDBRolesRepository(mongodb)
    roleService := rbac.NewRoleServiceDefault(rolesRepository)
    permissionService := rbac.NewPermissionServiceDefault(permissionsRepository)
    rbacHandler := rbacapi.NewRbacHandler(roleService, permissionService)
    
    rbacapi.RbacRoutesV1(tenantGroup, rbacHandler)
    
    // Setup account RBAC
    accountsRBAC := accountsRbac.NewAccountRBACServiceDefault(accountRepository, permissionService, roleService)
    accountsRbacHandler := accountsrbacapi.NewAccountRBACHandler(accountsRBAC)
    accountsrbacapi.AccountRBACRoutesV1(tenantGroup, accountsRbacHandler)
    
    // Setup tenants routes (might need different auth?)
    tenantsHandler := tenantsapi.NewTenantHandler(tenantService)
    tenantsapi.TenantRoutesV1(service, tenantsHandler)  // Note: Not on tenantGroup
    
    // Setup admin account service
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
    
    // Health check (no auth required)
    healthHandler := health.NewHealthHandler()
    health.HealthRoutes(service, healthHandler)
    
    // CORS must be set before routes
    corsSetting(service, config, logger)
    
    // Start server
    if err := service.Start(fmt.Sprintf(":%d", config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.Error(err.Error())
    }
}
```

## Middleware Execution Order

```
HTTP Request: GET /api/v1/tenant/tenant-123/accounts
Authorization: Bearer <jwt>
         ↓
┌────────────────────────────────────────┐
│ 1. Route Matching                      │
│    Route: /api/v1/tenant/:tenantid/*   │
│    Param: tenantid = "tenant-123"      │
└────────────────────────────────────────┘
         ↓
┌────────────────────────────────────────┐
│ 2. JWT Middleware                      │
│    - Extracts: c.Param("tenantid")     │
│      ✅ Returns: "tenant-123"          │
│    - Calls: ValidateAccessToken(       │
│        ctx,                             │
│        "tenant-123",                    │
│        jwt                              │
│      )                                  │
│    - Stores claims in context          │
└────────────────────────────────────────┘
         ↓
┌────────────────────────────────────────┐
│ 3. Tenant Middleware                   │
│    - Gets claims from context          │
│    - Validates tenant exists           │
│    - Checks authorization:             │
│      • System admin? → ALLOW           │
│      • claims.TenantID == "tenant-123"?│
│        → ALLOW, else DENY (403)        │
└────────────────────────────────────────┘
         ↓
┌────────────────────────────────────────┐
│ 4. Handler                             │
│    - GetTenantIDFromEcho(c)            │
│    - Process request                   │
└────────────────────────────────────────┘
```

## Update Route Registration Functions

Change from `*echo.Echo` to `*echo.Group`:

### accounts/account_routes.go

```go
// OLD
func AccountRoutesV1(e *echo.Echo, handler *AccountHandler) {
    e.POST("/api/v1/tenant/:tenantid/accounts", handler.RegisterAccount)
    // ...
}

// NEW
func AccountRoutesV1(g *echo.Group, handler *AccountHandler) {
    g.POST("/accounts", handler.RegisterAccount)
    g.GET("/accounts/:id", handler.GetAccount)
    g.GET("/accounts", handler.ListAccounts)
    g.PUT("/accounts/email", handler.ChangeAccountEmail)
    g.PUT("/accounts/disable", handler.DisableAccount)
    g.PUT("/accounts/enable", handler.EnableAccount)
    g.PUT("/accounts/deactivate", handler.DeactivateAccount)
    g.PUT("/accounts/unlink", handler.UnlinkSocial)
}
```

### rbac/rbac_routes.go

```go
// OLD
func RbacRoutesV1(e *echo.Echo, handler *RbacHandler) {
    e.POST("/api/v1/rbac/tenant/:tenantid/roles", handler.CreateRole)
    // ...
}

// NEW
func RbacRoutesV1(g *echo.Group, handler *RbacHandler) {
    g.POST("/roles", handler.CreateRole)
    g.GET("/roles/:id", handler.GetRole)
    g.GET("/roles", handler.ListRoles)
    g.PUT("/roles/:id", handler.UpdateRole)
    g.DELETE("/roles/:id", handler.DeleteRole)
    g.PUT("/roles/:id/permissions", handler.AddPermissionToRole)
    g.DELETE("/roles/:id/permissions/:permissionId", handler.RemovePermissionFromRole)
    g.POST("/permissions", handler.CreatePermission)
    g.GET("/permissions/exists/:id", handler.DoesPermissionExist)
    g.GET("/permissions", handler.ListPermissions)
    g.DELETE("/permissions/:id", handler.DeletePermission)
}
```

### accounts/rbac/accounts_routes.go

```go
// Update similarly to use *echo.Group and relative paths
func AccountRBACRoutesV1(g *echo.Group, handler *AccountRBACHandler) {
    g.POST("/accounts/:id/roles", handler.AssignRole)
    g.DELETE("/accounts/:id/roles/:roleId", handler.RemoveRole)
    g.POST("/accounts/:id/permissions", handler.GrantPermission)
    g.DELETE("/accounts/:id/permissions/:permissionId", handler.RevokePermission)
}
```

## What About Tenant Management Routes?

For tenant management endpoints like `/api/v1/tenants`, you have options:

### Option 1: System Admin Only (Recommended)

```go
// System admin group - no tenantID in URL
systemAdminGroup := service.Group("/api/v1/admin")
systemAdminGroup.Use(jwt.Jwt)  // Wait, this won't work without tenantID!

// You'll need a special JWT middleware for system routes
systemAdminGroup.Use(jwtSystemMiddleware.JwtForSystemRoutes)
systemAdminGroup.Use(requireSystemAdmin)  // Custom middleware

tenantsapi.TenantRoutesV1(systemAdminGroup, tenantsHandler)
```

### Option 2: Within System Tenant Context

```go
// Tenants accessed via system tenant ID
tenantMgmtGroup := service.Group("/api/v1/tenant/:tenantid/admin/tenants")
tenantMgmtGroup.Use(jwt.Jwt)
tenantMgmtGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant)
tenantMgmtGroup.Use(requireSystemAdmin)  // Only allow bulwark_admin role

tenantsapi.TenantRoutesV1(tenantMgmtGroup, tenantsHandler)
```

### Option 3: No Authentication (Development Only)

```go
// WARNING: Only for local development
tenantsapi.TenantRoutesV1(service, tenantsHandler)
```

## Required Middleware for System Routes

You'll need to create a special JWT middleware for routes without tenantID:

```go
// api/middleware/jwt_system.go
func (jm JWTMiddleware) JwtForSystemRoutes(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        jwt := c.Request().Header.Get(echo.HeaderAuthorization)
        if jwt == "" {
            return echo.ErrUnauthorized
        }
        
        deviceId := c.Request().Header.Get("x-bulwark-device-id")
        systemTenantID := "00000000-0000-0000-0000-000000000000"
        
        jwt = strings.Replace(jwt, "Bearer ", "", 1)
        jwt = strings.TrimSpace(jwt)
        ctx := c.Request().Context()
        
        // Validate against system tenant
        claims, err := jm.auth.Authenticate.ValidateAccessToken(ctx, systemTenantID, jwt)
        if err != nil {
            return echo.NewHTTPError(http.StatusUnauthorized, problem.Details{
                Type:   "https://latebit.io/bulwark/errors/unauthorized",
                Title:  "Unauthorized",
                Status: http.StatusUnauthorized,
                Detail: "Invalid or expired token",
            })
        }
        
        c.Set("claims", authClaimsToAccountCLaims(claims, jwt, deviceId))
        return next(c)
    }
}
```

## Summary of Changes Needed

1. ✅ **JWT middleware code is correct** - No changes needed
2. ❌ **main.go needs update** - Move `jwt.Jwt` from global to route group
3. ❌ **Route files need update** - Change `*echo.Echo` to `*echo.Group`
4. ⚠️ **System routes need solution** - Decide on tenant management auth strategy
5. ⚠️ **Health endpoint** - Should remain unauthenticated or add auth?

## Testing the Fix

Once updated, test these scenarios:

1. **Valid tenant user accessing own tenant**
   - Should work: `GET /api/v1/tenant/tenant-123/accounts`
   - JWT contains: `tenantId: "tenant-123"`

2. **System admin accessing any tenant**
   - Should work: `GET /api/v1/tenant/tenant-456/accounts`
   - JWT contains: `tenantId: "00000000-0000-0000-0000-000000000000"`, `roles: ["bulwark_admin"]`

3. **Tenant user accessing another tenant**
   - Should fail with 403: `GET /api/v1/tenant/tenant-789/accounts`
   - JWT contains: `tenantId: "tenant-123"`

4. **Invalid tenant in URL**
   - Should fail with 404: `GET /api/v1/tenant/non-existent/accounts`

5. **Missing or invalid JWT**
   - Should fail with 401: Any protected route without Authorization header
