package tenants

import (
	"github.com/labstack/echo/v4"
)

// TenantRoutesV1 registers tenant management routes
// These routes should be protected by JWT + System Admin middleware
// Recommended usage:
//
//	adminGroup := service.Group("/api/v1/admin")
//	adminGroup.Use(jwt.JwtForSystemRoutes)
//	adminGroup.Use(middleware.RequireSystemAdmin)
//	TenantRoutesV1(adminGroup, handler)
func TenantRoutesV1(g *echo.Group, handler *TenantHandler) {
	g.POST("/tenants", handler.AddTenant)
	g.GET("/tenants", handler.ListTenants)
	g.GET("/tenants/:id", handler.GetTenant)
	g.PUT("/tenants/:id", handler.UpdateTenant)
	g.DELETE("/tenants/:id", handler.DeleteTenant)
}
