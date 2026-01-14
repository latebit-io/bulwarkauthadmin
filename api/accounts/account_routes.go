package accounts

import "github.com/labstack/echo/v4"

func AccountRoutesV1(e *echo.Echo, handler *AccountHandler) {
	e.POST("/api/v1/tenant/:tenantid/accounts", handler.RegisterAccount)
	e.GET("/api/v1/tenant/:tenantid/accounts/:id", handler.GetAccount)
	e.GET("/api/v1/tenant/:tenantid/accounts", handler.ListAccounts)
	e.PUT("/api/v1/tenant/:tenantid/accounts/email", handler.ChangeAccountEmail)
	e.PUT("/api/v1/tenant/:tenantid/accounts/disable", handler.DisableAccount)
	e.PUT("/api/v1/tenant/:tenantid/accounts/enable", handler.EnableAccount)
	e.PUT("/api/v1/tenant/:tenantid/accounts/deactivate", handler.DeactivateAccount)
	e.PUT("/api/v1/tenant/:tenantid/accounts/unlink", handler.UnlinkSocial)
}
