# BulwarkAuthAdmin API Reference

Complete REST API documentation for the BulwarkAuthAdmin service.

## Quick Start

### View the OpenAPI Specification

The complete API is documented in `openapi.yaml` (OpenAPI 3.0 format).

**Online viewers:**
- **Swagger UI:** `https://editor.swagger.io/` → File → Load openapi.yaml
- **Redoc:** `https://redoc.ly/` → Upload openapi.yaml
- **Visual Studio Code:** Use the OpenAPI extension to view inline

### Base URL

```
http://localhost:8081/api/v1
```

### Authentication

All endpoints (except `/health`) require a **Bearer JWT token** from BulwarkAuth:

```bash
Authorization: Bearer <jwt_token>
```

## Authorization Model

### Role-Based Access Control

The API uses a hierarchical authorization model:

#### System Admin (`bulwark_admin`)
- Located in system tenant (`00000000-0000-0000-0000-000000000000`)
- Can manage all tenants
- Routes: `/api/v1/admin/tenants`
- Implicitly has tenant admin access for all tenants

#### Tenant Admin (`tenant_admin`)
- Located in their specific tenant
- Can manage accounts and RBAC within their tenant
- Routes: `/api/v1/tenant/:tenantId/accounts`, `/api/v1/tenant/:tenantId/rbac`, etc.
- Cannot access other tenants
- Created automatically for each tenant

#### Regular User
- Can perform operations based on assigned permissions
- Cannot access tenant management endpoints

### Default Permissions per Tenant

When a tenant is created, the following are created automatically:

| Permission | Description |
|-----------|-------------|
| `accounts:manage` | Manage accounts in the tenant |
| `rbac:manage` | Manage roles and permissions |

**Default Role:** `tenant_admin` (has both permissions above)

## API Endpoints Summary

### Health Check
- `GET /health` - Service health check (no auth required)

### Tenant Management (System Admin Only)
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/admin/tenants` | Create tenant |
| `GET` | `/admin/tenants` | List all tenants |
| `GET` | `/admin/tenants/{tenantId}` | Get tenant details |
| `PUT` | `/admin/tenants/{tenantId}` | Update tenant |
| `DELETE` | `/admin/tenants/{tenantId}` | Delete tenant |

### Account Management (Tenant Admin Required)
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/tenant/{tenantId}/accounts` | Create account |
| `GET` | `/tenant/{tenantId}/accounts` | List accounts |
| `GET` | `/tenant/{tenantId}/accounts/{accountId}` | Get account details |
| `PUT` | `/tenant/{tenantId}/accounts/email` | Change account email |
| `PUT` | `/tenant/{tenantId}/accounts/enable` | Enable account |
| `PUT` | `/tenant/{tenantId}/accounts/disable` | Disable account |
| `PUT` | `/tenant/{tenantId}/accounts/deactivate` | Soft delete account |
| `PUT` | `/tenant/{tenantId}/accounts/unlink` | Unlink social provider |

### RBAC Management (Tenant Admin Required)
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/tenant/{tenantId}/rbac/roles` | Create role |
| `GET` | `/tenant/{tenantId}/rbac/roles` | List roles |
| `GET` | `/tenant/{tenantId}/rbac/roles/{roleName}` | Get role details |
| `PUT` | `/tenant/{tenantId}/rbac/roles/{roleName}` | Update role |
| `DELETE` | `/tenant/{tenantId}/rbac/roles/{roleName}` | Delete role |
| `POST` | `/tenant/{tenantId}/rbac/permissions` | Create permission |
| `GET` | `/tenant/{tenantId}/rbac/permissions` | List permissions |
| `DELETE` | `/tenant/{tenantId}/rbac/permissions/{permissionKey}` | Delete permission |

### Account RBAC (Tenant Admin Required)
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/tenant/{tenantId}/accounts/rbac/roles` | Assign role to account |
| `DELETE` | `/tenant/{tenantId}/accounts/rbac/roles` | Remove role from account |
| `POST` | `/tenant/{tenantId}/accounts/rbac/permissions` | Assign permission to account |
| `DELETE` | `/tenant/{tenantId}/accounts/rbac/permissions` | Remove permission from account |

## Common Workflows

### Create and Setup a New Tenant

```bash
# 1. Create tenant (system admin)
curl -X POST http://localhost:8081/api/v1/admin/tenants \
  -H "Authorization: Bearer <system_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "Acme Corp",
    "Description": "Acme Corporation",
    "Domain": "acme.example.com"
  }'
# Response: { "id": "tenant-123", "name": "Acme Corp", ... }

# 2. Create tenant admin account (tenant admin or system admin)
curl -X POST http://localhost:8081/api/v1/tenant/tenant-123/accounts \
  -H "Authorization: Bearer <system_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{ "Email": "admin@acme.com" }'
# Response: { "id": "account-456", "email": "admin@acme.com", ... }

# 3. Assign tenant_admin role (system admin)
curl -X POST http://localhost:8081/api/v1/tenant/tenant-123/accounts/rbac/roles \
  -H "Authorization: Bearer <system_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": "account-456",
    "role": "tenant_admin"
  }'
```

### Manage Accounts as Tenant Admin

```bash
# Create account
curl -X POST http://localhost:8081/api/v1/tenant/tenant-123/accounts \
  -H "Authorization: Bearer <tenant_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{ "Email": "user@acme.com" }'

# List accounts
curl -X GET http://localhost:8081/api/v1/tenant/tenant-123/accounts \
  -H "Authorization: Bearer <tenant_admin_token>"

# Disable account
curl -X PUT http://localhost:8081/api/v1/tenant/tenant-123/accounts/disable \
  -H "Authorization: Bearer <tenant_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{ "AccountId": "account-789" }'
```

### Create Custom Roles and Permissions

```bash
# Create custom permission
curl -X POST http://localhost:8081/api/v1/tenant/tenant-123/rbac/permissions \
  -H "Authorization: Bearer <tenant_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "reports",
    "Action": "generate"
  }'
# Response: { "key": "reports:generate", ... }

# Create custom role
curl -X POST http://localhost:8081/api/v1/tenant/tenant-123/rbac/roles \
  -H "Authorization: Bearer <tenant_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "Name": "report_generator",
    "Description": "Can generate reports"
  }'

# Assign permission to role
curl -X PUT http://localhost:8081/api/v1/tenant/tenant-123/rbac/roles/report_generator/permissions \
  -H "Authorization: Bearer <tenant_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{ "Permission": "reports:generate" }'

# Assign role to user
curl -X POST http://localhost:8081/api/v1/tenant/tenant-123/accounts/rbac/roles \
  -H "Authorization: Bearer <tenant_admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "accountId": "account-user",
    "role": "report_generator"
  }'
```

## Response Format

### Success Response

All successful responses use standard HTTP status codes and return JSON:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Example",
  "created": "2024-01-22T10:00:00Z"
}
```

### Error Response (RFC 7807 Problem Details)

```json
{
  "type": "https://latebit.io/bulwark/errors/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "The requested resource does not exist"
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (success with no response body) |
| 400 | Bad Request |
| 401 | Unauthorized (missing or invalid token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found |
| 409 | Conflict (e.g., email already exists) |
| 500 | Internal Server Error |

## Pagination

Endpoints that return lists support pagination:

```bash
curl -X GET "http://localhost:8081/api/v1/tenant/tenant-123/accounts?page=0&size=10" \
  -H "Authorization: Bearer <token>"
```

**Query Parameters:**
- `page` - Page number (0-indexed, default: 0)
- `size` - Page size (default: 10)

## Tenant Isolation

All tenant-scoped endpoints enforce tenant isolation:

```bash
# ✅ User from tenant-123 can access their own tenant
GET /api/v1/tenant/tenant-123/accounts

# ❌ User from tenant-123 cannot access tenant-456
GET /api/v1/tenant/tenant-456/accounts
# Returns: 403 Forbidden
```

## System Tenant

The system tenant contains only system admins:

**System Tenant ID:** `00000000-0000-0000-0000-000000000000`

- Created automatically on service startup
- Contains `bulwark_admin` role and `bulwark_admin:write` permission
- System admin accounts have access to `/api/v1/admin/tenants` endpoints
- Cannot be deleted

## Testing the API

### Using curl

```bash
# Test health check
curl http://localhost:8081/api/v1/health

# List tenants (requires system admin token)
curl -H "Authorization: Bearer <system_admin_token>" \
  http://localhost:8081/api/v1/admin/tenants
```

### Using Postman

1. Import `openapi.yaml` into Postman
2. Set environment variable: `baseUrl=http://localhost:8081/api/v1`
3. Set authorization header in pre-request script or individual requests
4. Test endpoints with auto-generated examples

### Using Integration Tests

```bash
# Run integration tests
./run-integration-tests.sh

# Run specific test
go test -v -tags=integration -run TestAccountHandler_RegisterAccount ./tests/integration/accounts
```

## See Also

- **OpenAPI Spec:** `openapi.yaml` - Complete formal specification
- **CLAUDE.md:** Architecture and code patterns
- **Code Examples:** See `tests/integration/` for realistic usage examples
