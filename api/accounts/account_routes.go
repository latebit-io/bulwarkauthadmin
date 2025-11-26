package accounts

import "github.com/labstack/echo/v4"

func AccountRoutes(e *echo.Echo, handler AccountHandler) {
	e.POST("/api/accounts", handler.RegisterAccount)
	e.GET("/api/accounts/:id", handler.GetAccount)
	e.GET("/api/accounts", handler.ListAccounts)
	e.PUT("/api/accounts/email", handler.ChangeAccountEmail)
	e.PUT("/api/accounts/disable", handler.DisableAccount)
	e.PUT("/api/accounts/enable", handler.EnableAccount)
	e.PUT("/api/accounts/deactivate", handler.DeactivateAccount)

}
