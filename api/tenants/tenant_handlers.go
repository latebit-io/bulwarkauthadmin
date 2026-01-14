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
