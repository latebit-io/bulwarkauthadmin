package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/latebit-io/bulwarkauthadmin/api/problem"
)

// RequireSystemAdmin is middleware that ensures the authenticated user has the system admin role
// This middleware must run AFTER JWT middleware (which sets the claims)
func RequireSystemAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get claims from context (set by JWT middleware)
		claims, ok := GetAccountClaims(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/unauthorized",
				Title:  "Unauthorized",
				Status: http.StatusUnauthorized,
				Detail: "Authentication required",
			})
		}

		// Check if user is system admin
		if !IsSystemAdmin(claims) {
			return echo.NewHTTPError(http.StatusForbidden, problem.Details{
				Type:   "https://latebit.io/bulwark/errors/forbidden",
				Title:  "Access Denied",
				Status: http.StatusForbidden,
				Detail: "System administrator access required",
			})
		}

		return next(c)
	}
}
