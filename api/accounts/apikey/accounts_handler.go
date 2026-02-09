package apikey

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/middleware"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts/apikey"
)

type NewApiKeyRequest struct {
	Name string `json:"name"`
}

type AccountApiKeyHandler struct {
	apikey apikey.ApiKeyService
}

func NewAccountApiKeyHandler(apikey apikey.ApiKeyService) *AccountApiKeyHandler {
	return &AccountApiKeyHandler{
		apikey: apikey,
	}
}

func (h *AccountApiKeyHandler) CreateApiKey(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromEcho(c)
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	newApiRequest := new(NewApiKeyRequest)
	err := c.Bind(newApiRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	if newApiRequest.Name == "" {
		httpError := problem.NewBadRequest(errors.New("name is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	apiKey, err := h.apikey.Generate(ctx, tenantID, accountID, newApiRequest.Name, nil)
	if err != nil {
		var apiKeyDuplicateError accounts.ApiKeyDuplicateError
		duplicate := errors.As(err, &apiKeyDuplicateError)
		if duplicate {
			return echo.NewHTTPError(http.StatusConflict, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Duplicate Api Key",
				Status: http.StatusConflict,
				Detail: err.Error(),
			})
		}
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusCreated, apiKey)
}

func (h *AccountApiKeyHandler) RevokeApiKey(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromEcho(c)
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	apiKeyID := c.Param("apiKeyID")
	if apiKeyID == "" {
		httpError := problem.NewBadRequest(errors.New("api key is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err := h.apikey.Revoke(ctx, apiKeyID, tenantID, accountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *AccountApiKeyHandler) SuspendApiKey(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromEcho(c)
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	apiKeyID := c.Param("apiKeyID")
	if apiKeyID == "" {
		httpError := problem.NewBadRequest(errors.New("api key is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err := h.apikey.Suspend(ctx, apiKeyID, tenantID, accountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *AccountApiKeyHandler) EnableApiKey(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromEcho(c)
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	apiKeyID := c.Param("apiKeyID")
	if apiKeyID == "" {
		httpError := problem.NewBadRequest(errors.New("api key is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err := h.apikey.Enable(ctx, apiKeyID, tenantID, accountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *AccountApiKeyHandler) ListApiKeys(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromEcho(c)
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	keys, err := h.apikey.List(ctx, tenantID, accountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, keys)
}

func (h *AccountApiKeyHandler) GetApiKey(c echo.Context) error {
	tenantID := middleware.GetTenantIDFromEcho(c)
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	apiKeyID := c.Param("apiKeyID")
	if apiKeyID == "" {
		httpError := problem.NewBadRequest(errors.New("api key is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	key, err := h.apikey.GetKey(ctx, apiKeyID, tenantID, accountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, key)
}
