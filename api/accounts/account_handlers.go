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

type VerifyAccountRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password"`
}

type DeleteAccountRequest struct {
	Email       string `json:"email"`
	AccessToken string `json:"accessToken"`
}

type ChangePasswordRequest struct {
	Email       string `json:"email"`
	Password    string `json:"newPassword"`
	AccessToken string `json:"accessToken"`
}

type ChangeEmailRequest struct {
	Email     string `json:"email"`
	AccountID string `json:"accountId"`
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
