package accounts

import "github.com/labstack/echo/v4"

func AccountRoutes(e *echo.Echo, handler AccountHandler) {
	e.POST("/api/accounts", handler.RegisterAccount)
	e.GET("/api/accounts/:id", handler.GetAccount)
	e.PUT("/api/accounts/email", handler.ChangeAccountEmail)
}
