package rbac

import "github.com/labstack/echo/v4"

func AccountRBACRoutesV1(e *echo.Echo, handler *AccountRBACHandler) {
	e.PATCH("/api/v1/tenant/:tenantid/accounts/:id/rbac/role", handler.AssignRole)
	e.PATCH("/api/v1/tenant/:tenantid/accounts/:id/rbac/permission", handler.AssignPermission)
	e.DELETE("/api/v1/tenant/:tenantid/accounts/:id/rbac/role/:role", handler.RemoveRole)
	e.DELETE("/api/v1/tenant/:tenantid/accounts/:id/rbac/permission/:permission", handler.RemovePermission)
}
