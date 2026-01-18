# Tenant Authorization - Quick Reference

## Authorization Rules

| User Role | Tenant Access |
|-----------|---------------|
| System Admin (`bulwark_admin`) | ✅ All tenants (including system tenant) |
| Tenant Admin/User | ✅ Only their assigned tenant (JWT tenantID must match URL tenantID) |

## Middleware Order (CRITICAL)

```go
// 1. JWT Middleware FIRST (sets claims in context)
service.Use(jwt.Jwt)

// 2. Tenant Middleware SECOND (reads claims, validates tenant access)
tenantGroup := service.Group("/api/v1/tenant/:tenantid")
tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant)
```

## Constants

```go
const (
    SystemAdminRole = "bulwark_admin"
    SystemTenantID  = "00000000-0000-0000-0000-000000000000"
)
```

## Functions

### `ExtractAndAuthorizeTenant(next) HandlerFunc`
**Use this for most routes** - Validates tenant exists AND checks user access.

Returns:
- `400` - Tenant ID missing from URL
- `401` - User not authenticated
- `403` - User lacks access to tenant
- `404` - Tenant doesn't exist
- `500` - Database error

### `ExtractTenant(next) HandlerFunc`
Only validates tenant exists. Use when you need custom authorization logic in handler.

### `IsSystemAdmin(claims) bool`
Checks if user has system admin role.

### `CanAccessTenant(claims, tenantID) bool`
Checks if user can access a specific tenant.

### `GetTenantIDFromEcho(c) string`
Gets validated tenant ID from context. Always use this in handlers.

### `GetAccountClaims(c) (AccountClaims, bool)`
Gets JWT claims from context. Returns claims and ok boolean.

## Handler Pattern

```go
func (h *Handler) SomeEndpoint(c echo.Context) error {
    // Get validated tenant ID (already authorized by middleware)
    tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)
    
    // Get user claims if needed
    claims, ok := bulwarkauthmiddleware.GetAccountClaims(c)
    if !ok {
        return echo.ErrUnauthorized
    }
    
    // Optional: Additional authorization checks
    if needsSpecialPermission && !claims.HasPermission("special") {
        return echo.ErrForbidden
    }
    
    // Use tenantID in service calls
    ctx := c.Request().Context()
    result, err := h.service.DoSomething(ctx, tenantID, params...)
    
    return c.JSON(http.StatusOK, result)
}
```

## Security Notes

1. **JWT middleware MUST run before tenant middleware** - Tenant middleware depends on claims
2. **System tenant is special** - Use `SystemTenantID` constant (UUID nil)
3. **All database operations must filter by tenant** - Prevents data leakage
4. **System admins bypass tenant restrictions** - By design, for platform administration
5. **JWT tenantID is source of truth** - URL tenantID is what they're requesting access to

## Testing Checklist

- [ ] System admin can access any tenant
- [ ] Regular user can access their own tenant
- [ ] Regular user CANNOT access other tenants (returns 403)
- [ ] Invalid tenant returns 404
- [ ] Missing JWT claims returns 401
- [ ] All service methods receive and use tenantID parameter
- [ ] All database queries filter by tenantID
- [ ] Database indexes include tenantID as first field

## Common Mistakes

❌ **Forgetting to extract tenant ID in handlers**
```go
// WRONG - not using tenantID
account, err := h.service.GetAccount(ctx, accountID)
```

✅ **Correct**
```go
// RIGHT - always pass tenantID
tenantID := bulwarkauthmiddleware.GetTenantIDFromEcho(c)
account, err := h.service.GetAccount(ctx, tenantID, accountID)
```

❌ **Running tenant middleware before JWT middleware**
```go
// WRONG ORDER
tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant)
service.Use(jwt.Jwt)  // Too late!
```

✅ **Correct order**
```go
// RIGHT ORDER
service.Use(jwt.Jwt)  // First: authenticate and set claims
tenantGroup.Use(tenantMiddleware.ExtractAndAuthorizeTenant)  // Second: authorize tenant access
```

❌ **Not filtering database queries by tenant**
```go
// WRONG - security vulnerability!
filter := bson.M{"email": email}
```

✅ **Correct**
```go
// RIGHT - always include tenantID
filter := bson.M{"tenantId": tenantID, "email": email}
```

## Migration Guide

When adding tenant isolation to existing code:

1. Add `TenantID string` field to all models
2. Update all service interfaces to accept `tenantID string` as first parameter
3. Update all repository methods to accept and filter by `tenantID`
4. Add database indexes: `{tenantId: 1, otherFields...}`
5. Update all MongoDB queries to include `tenantId` in filter
6. Update route groups to use tenant middleware
7. Update all handlers to extract and pass tenantID
8. Write integration tests for tenant isolation
