package rbac

import "github.com/labstack/echo/v4"

func RbacRoutesV1(e *echo.Group, handler *RbacHandler) {
	rbacGroup := e.Group("/rbac")
	rbacGroup.POST("/roles", handler.CreateRole)
	rbacGroup.GET("/roles/:id", handler.GetRole)
	rbacGroup.GET("/roles", handler.ListRoles)
	rbacGroup.PUT("/roles/:id", handler.UpdateRole)
	rbacGroup.DELETE("/roles/:id", handler.DeleteRole)
	rbacGroup.PUT("/roles/:id/permissions", handler.AddPermissionToRole)
	rbacGroup.DELETE("/roles/:id/permissions/:permissionId", handler.RemovePermissionFromRole)
	rbacGroup.POST("/permissions", handler.CreatePermission)
	rbacGroup.GET("/permissions/exists/:id", handler.DoesPermissionExist)
	rbacGroup.GET("/permissions", handler.ListPermissions)
	rbacGroup.DELETE("/permissions/:id", handler.DeletePermission)
}
