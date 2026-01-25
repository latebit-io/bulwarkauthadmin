package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwark-auth-guard"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
)

type AccountClaims struct {
	Roles       []string  `json:"roles"`
	Issuer      string    `json:"issuer"`
	Subject     string    `json:"subject"`
	Audience    string    `json:"audience"`
	ExpiresAt   time.Time `json:"expiresAt"`
	NotBefore   time.Time `json:"notBefore"`
	IssuedAt    time.Time `json:"issuedAt"`
	ID          string    `json:"Id,omitempty"`
	AccessToken string    `json:"accessToken"`
	ClientID    string    `json:"clientId"`
	TenantID    string    `json:"tenantId"`
}

type JWTMiddleware struct {
	auth *bulwark.Guard
}

func NewJWTMiddleware(auth *bulwark.Guard) *JWTMiddleware {
	return &JWTMiddleware{auth: auth}
}

func (jm JWTMiddleware) Jwt(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		jwt := c.Request().Header.Get(echo.HeaderAuthorization)
		if jwt == "" {
			return echo.ErrUnauthorized
		}
		//user := c.Request().Header.Get("x-bulwark-account")
		deviceId := c.Request().Header.Get("x-bulwark-device-id")
		tenantID := c.Param("tenantid")

		jwt = strings.Replace(jwt, "Bearer ", "", 1)
		jwt = strings.TrimSpace(jwt)
		ctx := c.Request().Context()

		// Try to validate against the requested tenant first
		claims, err := jm.auth.Authenticate.ValidateAccessToken(ctx, tenantID, jwt)
		if err != nil {
			c.Logger().Warnf("Failed to validate JWT, trying system validation")
			systemTenantID := "00000000-0000-0000-0000-000000000000"
			systemClaims, systemErr := jm.auth.Authenticate.ValidateAccessToken(ctx, systemTenantID, jwt)
			if systemErr == nil && IsSystemAdmin(authClaimsToAccountClaims(systemClaims, jwt, deviceId)) {

				claims = systemClaims
				err = nil
			}
		}

		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, problem.NewBadRequest(err))
		}
		c.Set("claims", authClaimsToAccountClaims(claims, jwt, deviceId))
		return next(c)
	}
}

// JwtForSystemRoutes is middleware for routes without :tenantid parameter
// It validates JWT against the system tenant (00000000-0000-0000-0000-000000000000)
// Use this for admin-only routes like /api/v1/admin/*
func (jm JWTMiddleware) JwtForSystemRoutes(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		jwt := c.Request().Header.Get(echo.HeaderAuthorization)
		if jwt == "" {
			return echo.ErrUnauthorized
		}

		deviceId := c.Request().Header.Get("x-bulwark-device-id")

		// Use system tenant ID for validation
		systemTenantID := "00000000-0000-0000-0000-000000000000"

		jwt = strings.Replace(jwt, "Bearer ", "", 1)
		jwt = strings.TrimSpace(jwt)
		ctx := c.Request().Context()

		claims, err := jm.auth.Authenticate.ValidateAccessToken(ctx, systemTenantID, jwt)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/unauthorized",
				Title:  "Unauthorized",
				Status: http.StatusUnauthorized,
				Detail: "Invalid or expired token",
			})
		}

		c.Set("claims", authClaimsToAccountClaims(claims, jwt, deviceId))
		return next(c)
	}
}

func authClaimsToAccountClaims(claims bulwark.AccessTokenClaims, token, clientID string) AccountClaims {
	return AccountClaims{
		TenantID:    claims.TenantID,
		Roles:       claims.Roles,
		Issuer:      claims.Issuer,
		Subject:     claims.Subject,
		Audience:    claims.Audience,
		ExpiresAt:   claims.ExpiresAt,
		NotBefore:   claims.NotBefore,
		IssuedAt:    claims.IssuedAt,
		ClientID:    clientID,
		AccessToken: token,
	}
}
