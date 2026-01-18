package tenants

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
	"github.com/latebit-io/bulwarkauthadmin/internal/tenants"
)

type NewTenantRequest struct {
	Name        string
	Description string
	Domain      string
}

type TenantHandler struct {
	tenantService tenants.TenantService
}

func NewTenantHandler(tenantService tenants.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

func (t *TenantHandler) AddTenant(c echo.Context) error {
	newTenantRequest := &NewTenantRequest{}
	if err := c.Bind(newTenantRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	// Validate that Name and Description are not empty
	if newTenantRequest.Name == "" || newTenantRequest.Description == "" {
		httpError := problem.NewBadRequest(errors.New("tenant name and description are required and cannot be empty"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	if err := t.tenantService.AddTenant(ctx, newTenantRequest.Name, newTenantRequest.Description,
		newTenantRequest.Domain); err != nil {
		var tenantDuplicateError tenants.TenantDuplicateError
		duplicate := errors.As(err, &tenantDuplicateError)
		if duplicate {
			return echo.NewHTTPError(http.StatusConflict, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Duplicate Tenant",
				Status: http.StatusConflict,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusCreated)
}

func (t *TenantHandler) ListTenants(c echo.Context) error {
	ctx := c.Request().Context()
	tenants, err := t.tenantService.ListTenants(ctx)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, tenants)
}

func (t *TenantHandler) GetTenant(c echo.Context) error {
	tenantID := c.Param("id")
	if tenantID == "" {
		httpError := problem.NewBadRequest(errors.New("tenant ID is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	tenant, err := t.tenantService.GetTenant(ctx, tenantID)
	if err != nil {
		var tenantNotFoundError tenants.TenantNotFoundError
		notFound := errors.As(err, &tenantNotFoundError)
		if notFound {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Tenant Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, tenant)
}

func (t *TenantHandler) UpdateTenant(c echo.Context) error {
	tenantID := c.Param("id")
	if tenantID == "" {
		httpError := problem.NewBadRequest(errors.New("tenant ID is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	updateTenantRequest := &NewTenantRequest{}
	if err := c.Bind(updateTenantRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	// Validate that Name and Description are not empty
	if updateTenantRequest.Name == "" || updateTenantRequest.Description == "" {
		httpError := problem.NewBadRequest(errors.New("tenant name and description are required and cannot be empty"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	if err := t.tenantService.UpdateTenant(ctx, tenantID, updateTenantRequest.Name,
		updateTenantRequest.Description, updateTenantRequest.Domain); err != nil {
		var tenantNotFoundError tenants.TenantNotFoundError
		notFound := errors.As(err, &tenantNotFoundError)
		if notFound {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Tenant Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (t *TenantHandler) DeleteTenant(c echo.Context) error {
	tenantID := c.Param("id")
	if tenantID == "" {
		httpError := problem.NewBadRequest(errors.New("tenant ID is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	if err := t.tenantService.RemoveTenant(ctx, tenantID); err != nil {
		var tenantNotFoundError tenants.TenantNotFoundError
		notFound := errors.As(err, &tenantNotFoundError)
		if notFound {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Tenant Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}
