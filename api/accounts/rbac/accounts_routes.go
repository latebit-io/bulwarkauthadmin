package rbac

import "github.com/labstack/echo/v4"

func AccountRBACRoutesV1(e *echo.Group, handler *AccountRBACHandler) {
	e.PATCH("/accounts/:id/rbac/role", handler.AssignRole)
	e.PATCH("/accounts/:id/rbac/permission", handler.AssignPermission)
	e.DELETE("/accounts/:id/rbac/role/:role", handler.RemoveRole)
	e.DELETE("/accounts/:id/rbac/permission/:permission", handler.RemovePermission)
}
