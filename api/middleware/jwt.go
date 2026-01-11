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
		user := c.Request().Header.Get("x-bulwark-account")
		deviceId := c.Request().Header.Get("x-bulwark-device-id")

		jwt = strings.Replace(jwt, "Bearer ", "", 1)
		jwt = strings.TrimSpace(jwt)
		ctx := c.Request().Context()
		claims, err := jm.auth.Authenticate.ValidateAccessToken(ctx, user, jwt, deviceId)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, problem.NewBadRequest(err))
		}
		c.Set("claims", authClaimsToAccountCLaims(claims, jwt, deviceId))
		return next(c)
	}
}

func authClaimsToAccountCLaims(claims bulwark.AccessTokenClaims, token, clientID string) AccountClaims {
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
