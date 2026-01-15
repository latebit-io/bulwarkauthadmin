package rbac

import "github.com/labstack/echo/v4"

func RbacRoutesV1(e *echo.Group, handler *RbacHandler) {
	e.POST("/roles", handler.CreateRole)
	e.GET("/roles/:id", handler.GetRole)
	e.GET("/roles", handler.ListRoles)
	e.PUT("/roles/:id", handler.UpdateRole)
	e.DELETE("/roles/:id", handler.DeleteRole)
	e.PUT("/roles/:id/permissions", handler.AddPermissionToRole)
	e.DELETE("/roles/:id/permissions/:permissionId", handler.RemovePermissionFromRole)
	e.POST("/permissions", handler.CreatePermission)
	e.GET("/permissions/exists/:id", handler.DoesPermissionExist)
	e.GET("/permissions", handler.ListPermissions)
	e.DELETE("/permissions/:id", handler.DeletePermission)
}
