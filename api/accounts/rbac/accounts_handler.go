package rbac

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts/rbac"
)

type AssignRoleRequest struct {
	AccountID string `json:"accountId"`
	Role      string `json:"role"`
}

type RemoveRoleRequest struct {
	AccountID string `json:"accountId"`
	Role      string `json:"role"`
}

type AssignPermissionRequest struct {
	AccountID  string `json:"accountId"`
	Permission string `json:"permission"`
}

type RemovePermissionRequest struct {
	AccountID  string `json:"accountId"`
	Permission string `json:"permission"`
}

type AccountRBACHandler struct {
	accountRbacService rbac.AccountRBACService
}

func NewAccountRBACHandler(accountRbacService rbac.AccountRBACService) *AccountRBACHandler {
	return &AccountRBACHandler{
		accountRbacService: accountRbacService,
	}
}

func (ah *AccountRBACHandler) AssignRole(c echo.Context) error {
	assignRoleRequest := new(AssignRoleRequest)
	err := c.Bind(assignRoleRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accountRbacService.AssignRole(ctx, assignRoleRequest.AccountID, assignRoleRequest.Role)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountRBACHandler) RemoveRole(c echo.Context) error {
	removeRoleRequest := new(RemoveRoleRequest)
	err := c.Bind(removeRoleRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accountRbacService.RemoveRole(ctx, removeRoleRequest.AccountID, removeRoleRequest.Role)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountRBACHandler) AssignPermission(c echo.Context) error {
	assignPermissionRequest := new(AssignPermissionRequest)
	err := c.Bind(assignPermissionRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accountRbacService.AssignPermission(ctx, assignPermissionRequest.AccountID, assignPermissionRequest.Permission)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountRBACHandler) RemovePermission(c echo.Context) error {
	removePermissionRequest := new(RemovePermissionRequest)
	err := c.Bind(removePermissionRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accountRbacService.RemovePermission(ctx, removePermissionRequest.AccountID, removePermissionRequest.Permission)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}
