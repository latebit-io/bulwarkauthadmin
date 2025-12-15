package rbac

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
	"github.com/latebit-io/bulwarkauthadmin/internal/rbac"
	"github.com/latebit-io/bulwarkauthadmin/internal/shared"
)

type NewRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DeleteRoleRequest struct {
	Name string `json:"name"`
}

type GetRoleRequest struct {
	Name string `json:"name"`
}

type ListRolesRequest struct {
	Size int `json:"limit"`
	Page int `json:"offset"`
}

type AddPermissionRoleRequest struct {
	RoleName      string `json:"roleName"`
	PermissionKey string `json:"permissionKey"`
}

type RemovePermissionRoleRequest struct {
	RoleName      string `json:"roleName"`
	PermissionKey string `json:"permissionKey"`
}

type RbacHandler struct {
	roleServices       rbac.RoleService
	permissionServices rbac.PermissionService
}

func NewRbacHandler(roleServices rbac.RoleService, permissionServices rbac.PermissionService) RbacHandler {
	return RbacHandler{
		roleServices:       roleServices,
		permissionServices: permissionServices,
	}
}

type NewPermissionRequest struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type DeletePermissionRequest struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

type ListPermissionsRequest struct {
	shared.PageOptions
}

type DoesPermissionExistRequest struct {
	Key string `json:"key"`
}

func (r *RbacHandler) DoesPermissionExist(c echo.Context) error {
	permissionExistRequest := &DoesPermissionExistRequest{}
	if err := c.Bind(permissionExistRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()

	exists, err := r.permissionServices.DoesPermissionExist(ctx, permissionExistRequest.Key)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, exists)
}

func (r *RbacHandler) ListPermissions(c echo.Context) error {
	listPermissionsRequest := &ListPermissionsRequest{}
	if err := c.Bind(listPermissionsRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()

	permissions, err := r.permissionServices.ListPermissions(ctx, listPermissionsRequest.PageOptions)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, permissions)
}

func (r *RbacHandler) DeletePermission(c echo.Context) error {
	permissionKey := c.Param("id")
	if permissionKey == "" {
		httpError := problem.NewBadRequest(errors.New("permission key is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()

	// Permission key format is "name:action", need to parse it
	deletePermissionRequest := &DeletePermissionRequest{}
	if err := c.Bind(deletePermissionRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	if err := r.permissionServices.DeletePermission(ctx, deletePermissionRequest.Name, deletePermissionRequest.Action); err != nil {
		var permissionNotFoundError rbac.PermissionNotFoundError
		notFound := errors.As(err, &permissionNotFoundError)
		if notFound {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Permission Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (r *RbacHandler) CreatePermission(c echo.Context) error {
	newPermissionRequest := &NewPermissionRequest{}
	if err := c.Bind(newPermissionRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	if err := r.permissionServices.CreatePermission(ctx, newPermissionRequest.Name, newPermissionRequest.Action); err != nil {
		var permissionDuplicateError rbac.PermissionDuplicateError
		duplicate := errors.As(err, &permissionDuplicateError)
		if duplicate {
			return echo.NewHTTPError(http.StatusConflict, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Duplicate Permission",
				Status: http.StatusConflict,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusCreated)
}

func (r *RbacHandler) CreateRole(c echo.Context) error {
	newRoleRequest := &NewRoleRequest{}
	if err := c.Bind(newRoleRequest); err != nil {
		return err
	}
	ctx := c.Request().Context()
	if err := r.roleServices.CreateRole(ctx, newRoleRequest.Name, newRoleRequest.Description); err != nil {
		var roleDuplicateError rbac.RoleDuplicateError
		duplicate := errors.As(err, &roleDuplicateError)
		if duplicate {
			return echo.NewHTTPError(http.StatusConflict, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Duplicate Role",
				Status: http.StatusConflict,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusCreated)
}

func (r *RbacHandler) UpdateRole(c echo.Context) error {
	roleName := c.Param("id")
	if roleName == "" {
		httpError := problem.NewBadRequest(errors.New("role name is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	updateRoleRequest := &UpdateRoleRequest{}
	if err := c.Bind(updateRoleRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	if err := r.roleServices.UpdateRole(ctx, roleName, updateRoleRequest.Description); err != nil {
		var roleNotFoundError rbac.RoleNotFoundError
		notFound := errors.As(err, &roleNotFoundError)
		if notFound {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Role Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusCreated)
}

func (r *RbacHandler) DeleteRole(c echo.Context) error {
	roleName := c.Param("id")
	if roleName == "" {
		httpError := problem.NewBadRequest(errors.New("role name is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	if err := r.roleServices.DeleteRole(ctx, roleName); err != nil {
		var roleNotFoundError rbac.RoleNotFoundError
		notFound := errors.As(err, &roleNotFoundError)
		if notFound {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Role Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (r *RbacHandler) GetRole(c echo.Context) error {
	roleName := c.Param("id")
	if roleName == "" {
		httpError := problem.NewBadRequest(errors.New("role name is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	role, err := r.roleServices.GetRole(ctx, roleName)
	if err != nil {
		var roleNotFoundError rbac.RoleNotFoundError
		notFound := errors.As(err, &roleNotFoundError)
		if notFound {
			return echo.NewHTTPError(http.StatusNotFound, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Role Not Found",
				Status: http.StatusNotFound,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, role)
}

func (r *RbacHandler) ListRoles(c echo.Context) error {
	ctx := c.Request().Context()
	listRolesRequest := &ListRolesRequest{}
	if err := c.Bind(listRolesRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	pagingOptions := shared.NewPageOptions(listRolesRequest.Page, listRolesRequest.Size, "")
	roles, err := r.roleServices.ListRoles(ctx, pagingOptions)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, roles)
}

func (r *RbacHandler) AddPermissionToRole(c echo.Context) error {
	roleName := c.Param("id")
	if roleName == "" {
		httpError := problem.NewBadRequest(errors.New("role name is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	addPermissionRequest := &AddPermissionRoleRequest{}
	if err := c.Bind(addPermissionRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err := r.roleServices.AddPermission(ctx, roleName, addPermissionRequest.PermissionKey)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (r *RbacHandler) RemovePermissionFromRole(c echo.Context) error {
	roleName := c.Param("id")
	if roleName == "" {
		httpError := problem.NewBadRequest(errors.New("role name is required"))
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	removePermissionRequest := &RemovePermissionRoleRequest{}
	if err := c.Bind(removePermissionRequest); err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	err := r.roleServices.RemovePermission(ctx, roleName, removePermissionRequest.PermissionKey)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}
