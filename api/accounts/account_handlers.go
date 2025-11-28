package accounts

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
	"github.com/latebit-io/bulwarkauthadmin/internal/accounts"
)

type AccountHandler struct {
	accounts accounts.AccountManagementService
}

type NewAccountRequest struct {
	Email string `json:"email"`
}

type DisableAccountRequest struct {
	AccountID string `json:"accountId"`
}

type DeactivateAccountRequest struct {
	AccountID string `json:"accountId"`
}

type EnableAccountRequest struct {
	AccountID string `json:"accountId"`
}

type ChangeEmailRequest struct {
	Email     string `json:"email"`
	AccountID string `json:"accountId"`
}

type UnlinkSocialRequest struct {
	AccountID string `json:"accountId"`
	Provider  string `json:"provider"`
}

func NewAccountHandler(service accounts.AccountManagementService) AccountHandler {
	return AccountHandler{service}
}

// RegisterAccount handles the creation of a new account based on the provided email and password in the request payload.
func (ah AccountHandler) RegisterAccount(c echo.Context) error {
	newAccountRequest := new(NewAccountRequest)
	err := c.Bind(newAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err = ah.accounts.RegisterAccount(ctx, newAccountRequest.Email, accounts.AccountOptions{
		IsVerified: true,
	})
	if err != nil {
		var accountDuplicateError accounts.AccountDuplicateError
		duplicate := errors.As(err, &accountDuplicateError)
		if duplicate {
			return echo.NewHTTPError(http.StatusConflict, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Duplicate Account",
				Status: http.StatusConflict,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusCreated)
}

func (ah AccountHandler) GetAccount(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()
	account, err := ah.accounts.GetAccountDetails(ctx, id)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, account)
}

func (ah AccountHandler) ChangeAccountEmail(c echo.Context) error {
	changeEmailRequest := new(ChangeEmailRequest)
	err := c.Bind(changeEmailRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	err = ah.accounts.ChangeAccountEmail(ctx, changeEmailRequest.AccountID,
		changeEmailRequest.Email, accounts.AccountOptions{
			IsVerified: true,
		})
	if err != nil {
		var accountDuplicateError accounts.AccountDuplicateError
		duplicate := errors.As(err, &accountDuplicateError)
		if duplicate {
			return echo.NewHTTPError(http.StatusConflict, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/",
				Title:  "Email in use",
				Status: http.StatusConflict,
				Detail: err.Error(),
			})
		}

		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountHandler) ListAccounts(c echo.Context) error {
	accountsFilter := new(accounts.AccountFilter)
	err := c.Bind(accountsFilter)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	ctx := c.Request().Context()
	accounts, err := ah.accounts.ListAccounts(ctx, *accountsFilter)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.JSON(http.StatusOK, accounts)
}

func (ah *AccountHandler) DisableAccount(c echo.Context) error {
	disableAccountRequest := new(DisableAccountRequest)
	err := c.Bind(disableAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accounts.DisableAccount(ctx, disableAccountRequest.AccountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountHandler) EnableAccount(c echo.Context) error {
	enableAccountRequest := new(EnableAccountRequest)
	err := c.Bind(enableAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accounts.EnableAccount(ctx, enableAccountRequest.AccountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountHandler) DeactivateAccount(c echo.Context) error {
	deactivateAccountRequest := new(DeactivateAccountRequest)
	err := c.Bind(deactivateAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accounts.DeactivateAccount(ctx, deactivateAccountRequest.AccountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountHandler) PurgeAccount(c echo.Context) error {
	deactivateAccountRequest := new(DeactivateAccountRequest)
	err := c.Bind(deactivateAccountRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accounts.DeactivateAccount(ctx, deactivateAccountRequest.AccountID)
	if err != nil {
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (ah *AccountHandler) UnlinkSocial(c echo.Context) error {
	unlinkRequest := new(UnlinkSocialRequest)
	err := c.Bind(unlinkRequest)
	if err != nil {
		httpError := problem.NewBadRequest(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}
	ctx := c.Request().Context()
	err = ah.accounts.UnlinkSocialProvider(ctx, unlinkRequest.AccountID, unlinkRequest.Provider)
	if err != nil {
		var socialProviderNotFound accounts.SocialProviderNotFoundError
		notFound := errors.As(err, &socialProviderNotFound)
		if notFound {
			httpError := problem.NewBadRequest(err)
			return echo.NewHTTPError(httpError.Status, httpError)
		}
		httpError := problem.NewServerError(err)
		return echo.NewHTTPError(httpError.Status, httpError)
	}

	return c.NoContent(http.StatusNoContent)
}
