# BulwarkAuthAdmin Middleware Architecture

## Route Groups and Middleware Stack

```
┌─────────────────────────────────────────────────────────────────┐
│                    Echo Web Service                              │
└─────────────────────────────────────────────────────────────────┘
                              |
          ┌───────────────────┴───────────────────┐
          |                                       |
          v                                       v
┌──────────────────────┐              ┌──────────────────────┐
│   PUBLIC ROUTES      │              │   PROTECTED ROUTES   │
│   No Authentication  │              │   Require JWT        │
└──────────────────────┘              └──────────────────────┘
          |                                       |
          v                                       |
    /health (200 OK)         ┌───────────────────┴────────────────────┐
                             |                                        |
                             v                                        v
                ┌─────────────────────────┐            ┌─────────────────────────┐
                │  SYSTEM ADMIN ROUTES    │            │  TENANT-SCOPED ROUTES   │
                │  /api/v1/admin/*        │            │  /api/v1/tenant/:id/*   │
                └─────────────────────────┘            └─────────────────────────┘
                             |                                        |
                             v                                        v
                ┌─────────────────────────┐            ┌─────────────────────────┐
                │ 1. JwtForSystemRoutes   │            │ 1. Jwt (with :tenantid) │
                │    Validates with       │            │    Validates with URL   │
                │    system tenant ID     │            │    tenant parameter     │
                └─────────────────────────┘            └─────────────────────────┘
                             |                                        |
                             v                                        v
                ┌─────────────────────────┐            ┌─────────────────────────┐
                │ 2. RequireSystemAdmin   │            │ 2. ExtractAndAuthorize  │
                │    Checks bulwark_admin │            │    Validates tenant +   │
                │    role                 │            │    checks user access   │
                └─────────────────────────┘            └─────────────────────────┘
                             |                                        |
                             v                                        v
                ┌─────────────────────────┐            ┌─────────────────────────┐
                │   TENANT MANAGEMENT     │            │   TENANT RESOURCES      │
                │   - Create tenant       │            │   - Accounts            │
                │   - List tenants        │            │   - Roles               │
                │   - Update tenant       │            │   - Permissions         │
                │   - Delete tenant       │            │   - RBAC operations     │
                └─────────────────────────┘            └─────────────────────────┘
```

## Authorization Matrix

| Route Type | JWT Required | Tenant ID Source | Who Can Access | Middleware Stack |
|-----------|--------------|-----------------|----------------|------------------|
| **Public** (`/health`) | ❌ No | N/A | Anyone | None |
| **System Admin** (`/api/v1/admin/tenants`) | ✅ Yes | System tenant (hardcoded) | System admins only | `JwtForSystemRoutes` → `RequireSystemAdmin` |
| **Tenant-Scoped** (`/api/v1/tenant/:tenantid/*`) | ✅ Yes | URL parameter | System admins OR tenant members | `Jwt` → `ExtractAndAuthorizeTenant` |

## User Access Patterns

### System Administrator
```
JWT Claims:
  tenantId: "00000000-0000-0000-0000-000000000000"
  roles: ["bulwark_admin"]

Can Access:
  ✅ GET /api/v1/admin/tenants                    (Tenant management)
  ✅ POST /api/v1/admin/tenants                   (Create tenants)
  ✅ GET /api/v1/tenant/ANY-TENANT-ID/accounts    (All tenant resources)
  ✅ GET /api/v1/tenant/tenant-123/accounts       (Any specific tenant)
  ✅ GET /api/v1/tenant/tenant-456/roles          (Any specific tenant)
```

### Tenant Administrator/User
```
JWT Claims:
  tenantId: "tenant-123"
  roles: ["tenant_admin"]

Can Access:
  ❌ GET /api/v1/admin/tenants                    (403 - Not system admin)
  ❌ POST /api/v1/admin/tenants                   (403 - Not system admin)
  ✅ GET /api/v1/tenant/tenant-123/accounts       (Own tenant only)
  ✅ GET /api/v1/tenant/tenant-123/roles          (Own tenant only)
  ❌ GET /api/v1/tenant/tenant-456/accounts       (403 - Different tenant)
```

### Unauthenticated User
```
JWT Claims: None

Can Access:
  ✅ GET /health                                  (Public endpoint)
  ❌ GET /api/v1/admin/tenants                    (401 - No JWT)
  ❌ GET /api/v1/tenant/tenant-123/accounts       (401 - No JWT)
```

## Middleware Decision Tree

```
                        HTTP Request
                             |
                             v
                   [Does route need auth?]
                     |              |
                    NO              YES
                     |              |
                     v              v
              Handle Request   [Extract JWT]
                     |              |
                     v              v
                 200 OK    [Does route have :tenantid?]
                               |              |
                              NO              YES
                               |              |
                               v              v
                    [JwtForSystemRoutes]  [Jwt with :tenantid]
                    Validate with         Validate with
                    system tenant         URL tenant
                               |              |
                               v              v
                    [RequireSystemAdmin]  [ExtractAndAuthorizeTenant]
                    Check bulwark_admin   Check tenant access
                               |              |
                               v              v
                         [Has role?]    [Can access tenant?]
                          |        |      |              |
                         YES       NO    YES             NO
                          |        |      |              |
                          v        v      v              v
                      ALLOW      403   ALLOW           403
```

## Security Layers

### Layer 1: JWT Validation
- **Purpose:** Verify token authenticity and validity
- **Checks:**
  - Token signature is valid
  - Token hasn't expired
  - Token was issued for the correct tenant
- **Performed by:** `Jwt()` or `JwtForSystemRoutes()`

### Layer 2: Role-Based Authorization
- **Purpose:** Verify user has required role
- **Checks:**
  - User has `bulwark_admin` role (for admin routes)
- **Performed by:** `RequireSystemAdmin()`

### Layer 3: Tenant-Based Authorization
- **Purpose:** Verify user can access specific tenant
- **Checks:**
  - Tenant exists in database
  - User is system admin OR user's tenant matches requested tenant
- **Performed by:** `ExtractAndAuthorizeTenant()`

### Layer 4: Resource-Level Authorization (Future)
- **Purpose:** Verify user has permission for specific operation
- **Checks:**
  - User has required permission (e.g., "accounts:write")
  - Could be implemented in handlers or additional middleware

## Middleware Implementation Files

```
api/middleware/
├── jwt.go
│   ├── Jwt()                    - For tenant-scoped routes
│   └── JwtForSystemRoutes()     - For system admin routes
│
├── tenant_middleware.go
│   ├── ExtractTenant()                 - Validation only
│   ├── ExtractAndAuthorizeTenant()     - Validation + Authorization
│   ├── IsSystemAdmin()                 - Helper function
│   ├── CanAccessTenant()               - Helper function
│   └── GetTenantIDFromEcho()           - Context extraction
│
└── system_admin_middleware.go
    └── RequireSystemAdmin()     - Enforces bulwark_admin role
```

## Configuration in main.go

```go
// System admin routes (no tenant in URL)
adminGroup := service.Group("/api/v1/admin")
adminGroup.Use(jwt.JwtForSystemRoutes)
adminGroup.Use(bulwarkauthmiddleware.RequireSystemAdmin)
tenantsapi.TenantRoutesV1(adminGroup, tenantsHandler)

// Tenant-scoped routes (with :tenantid in URL)
tenantGroup := service.Group("/api/v1/tenant/:tenantid")
tenantGroup.Use(jwt.Jwt)
tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant)
accountsapi.AccountRoutesV1(tenantGroup, accountsHandler)
rbacapi.RbacRoutesV1(tenantGroup, rbacHandler)

// Public routes (no authentication)
health.HealthRoutes(service, healthHandler)
```

## Error Responses

| Status | Error | When | Middleware |
|--------|-------|------|-----------|
| 400 | Bad Request | Tenant ID missing from URL | `ExtractAndAuthorizeTenant` |
| 401 | Unauthorized | No JWT token or invalid token | `Jwt` / `JwtForSystemRoutes` |
| 403 | Forbidden | User lacks system admin role | `RequireSystemAdmin` |
| 403 | Forbidden | User can't access tenant | `ExtractAndAuthorizeTenant` |
| 404 | Not Found | Tenant doesn't exist | `ExtractAndAuthorizeTenant` |
| 500 | Server Error | Database error during validation | Any middleware |

## Best Practices

1. ✅ **Always apply JWT middleware before authorization middleware**
   - Authorization needs claims from JWT

2. ✅ **Use route groups for logical organization**
   - System admin routes separate from tenant routes

3. ✅ **Apply middleware in correct order**
   - Authentication → Authorization → Handler

4. ✅ **Use specific middleware for specific needs**
   - `JwtForSystemRoutes` for admin routes
   - `Jwt` for tenant-scoped routes

5. ✅ **Extract tenant ID using helper functions**
   - `GetTenantIDFromEcho(c)` in handlers
   - Never manually parse from URL

6. ✅ **Test authorization boundaries**
   - Verify users can't access unauthorized resources
   - Test system admin override works correctly

## Common Pitfalls to Avoid

❌ **Don't apply JWT middleware globally**
```go
// WRONG - JWT needs :tenantid parameter
service.Use(jwt.Jwt)
```

❌ **Don't skip tenant authorization**
```go
// WRONG - No tenant access check
tenantGroup.Use(jwt.Jwt)
// Missing: tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant)
```

❌ **Don't forget system admin middleware**
```go
// WRONG - Any authenticated user can manage tenants
adminGroup.Use(jwt.JwtForSystemRoutes)
// Missing: adminGroup.Use(RequireSystemAdmin)
```

❌ **Don't hardcode tenant IDs in handlers**
```go
// WRONG
tenantID := "some-hardcoded-value"

// RIGHT
tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)
```
