package rbac

import "github.com/labstack/echo/v4"

func RbacRoutesV1(e *echo.Echo, handler RbacHandler) {
	e.POST("/api/v1/rbac/roles", handler.CreateRole)
	e.GET("/api/v1/rbac/roles/:id", handler.GetRole)
	e.GET("/api/v1/rbac/roles", handler.ListRoles)
	e.PUT("/api/v1/rbac/roles/:id", handler.UpdateRole)
	e.DELETE("/api/v1/rbac/roles/:id", handler.DeleteRole)
	e.PUT("/api/v1/rbac/roles/:id/permissions", handler.AddPermissionToRole)
	e.DELETE("/api/v1/rbac/roles/:id/permissions", handler.RemovePermissionFromRole)
	e.POST("/api/v1/rbac/permissions", handler.CreatePermission)
	e.GET("/api/v1/rbac/permissions/exists/:id", handler.DoesPermissionExist)
	e.GET("/api/v1/rbac/permissions", handler.ListPermissions)
	e.DELETE("/api/v1/rbac/permissions/:id", handler.DeletePermission)
}
