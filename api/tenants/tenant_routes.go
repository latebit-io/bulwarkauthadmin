package tenants

import (
	"github.com/labstack/echo/v4"
)

func TenantRoutesV1(e *echo.Echo, handler *TenantHandler) {
	e.POST("/api/v1/tenant", handler.AddTenant)
}
