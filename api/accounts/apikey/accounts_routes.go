package apikey

import "github.com/labstack/echo/v4"

func AccountApiKeyRoutesV1(e *echo.Group, handler *AccountApiKeyHandler) {
	e.POST("/accounts/:id/apikeys", handler.CreateApiKey)
}
