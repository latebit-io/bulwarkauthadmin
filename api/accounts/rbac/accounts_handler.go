package rbac

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts/rbac"
)

type AssignRoleRequest struct {
	AccountID string `json:"accountId"`
	Role      string `json:"role"`
}

func (r AssignRoleRequest) Validate() error {
	if r.AccountID == "" {
		return errors.New("account id is required")
	}
	if r.Role == "" {
		return errors.New("role is required")
	}
	return nil
}

type AssignPermissionRequest struct {
	AccountID  string `json:"accountId"`
	Permission string `json:"permission"`
}

func (r AssignPermissionRequest) Validate() error {
	if r.AccountID == "" {
		return errors.New("account id is required")
	}
	if r.Permission == "" {
		return errors.New("permission is required")
	}
	return nil
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

	err = assignRoleRequest.Validate()
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
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	role := c.Param("role")
	if role == "" {
		httpError := problem.NewBadRequest(errors.New("role is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err := ah.accountRbacService.RemoveRole(ctx, accountID, role)
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
	err = assignPermissionRequest.Validate()
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
	accountID := c.Param("id")
	if accountID == "" {
		httpError := problem.NewBadRequest(errors.New("account is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	permission := c.Param("permission")
	if permission == "" {
		httpError := problem.NewBadRequest(errors.New("role is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err := ah.accountRbacService.RemovePermission(ctx, accountID, permission)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}
