package apikey

import "github.com/labstack/echo/v4"

func AccountApiKeyRoutesV1(e *echo.Group, handler *AccountApiKeyHandler) {
	e.POST("/accounts/:id/apikeys", handler.CreateApiKey)
	e.DELETE("/accounts/:id/apikeys/:apiKeyID", handler.RevokeApiKey)
	e.PUT("/accounts/:id/apikeys/:apiKeyID/enable", handler.EnableApiKey)
	e.PUT("/accounts/:id/apikeys/:apiKeyID/suspend", handler.SuspendApiKey)
	e.GET("/accounts/:id/apikeys", handler.ListApiKeys)
	e.GET("/accounts/:id/apikeys/:apiKeyID", handler.GetApiKey)
}
