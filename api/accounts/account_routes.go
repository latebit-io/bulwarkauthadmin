package accounts

import "github.com/labstack/echo/v4"

func AccountRoutesV1(e *echo.Group, handler *AccountHandler) {
	e.POST("/accounts", handler.RegisterAccount)
	e.GET("/accounts/:id", handler.GetAccount)
	e.GET("/accounts", handler.ListAccounts)
	e.PUT("/accounts/email", handler.ChangeAccountEmail)
	e.PUT("/accounts/disable", handler.DisableAccount)
	e.PUT("/accounts/enable", handler.EnableAccount)
	e.PUT("/accounts/deactivate", handler.DeactivateAccount)
	e.PUT("/accounts/unlink", handler.UnlinkSocial)
}
