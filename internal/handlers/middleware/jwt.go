package middleware

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/authscheme"
	slurperjwt "github.com/mailslurper/mailslurper/v2/pkg/auth/jwt"
	"github.com/mailslurper/mailslurper/v2/pkg/contexts"
)

func JWTAuth(
	config *io.Config,
	jwtService *slurperjwt.JWTService,
	logger *log.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if config.AuthenticationScheme == authscheme.NONE {
				next.ServeHTTP(writer, request.WithContext(AttachUser(request.Context(), "")))
			}

			logger.Print("Starting parse of JWT token")

			sToken := tokenFromHeader(request)
			if sToken == "" {
				err := fmt.Errorf("No bearer and token in authorization header")

				response.RenderOrLog(writer, request, response.HTTPUnauthorized(err), logger)

				return
			}

			var (
				token *jwt.Token
				err   error
			)

			if token, err = jwtService.Parse(sToken, config.AuthSecret); err != nil {
				err = fmt.Errorf("%w: Error parsing JWT token in service authorization middleware", err)

				response.RenderOrLog(writer, request, response.HTTPUnauthorized(err), logger)

				return
			}

			if err = jwtService.IsTokenValid(token); err != nil {
				err = fmt.Errorf("%w: Invalid token", err)

				response.RenderOrLog(writer, request, response.HTTPUnauthorized(err), logger)

				return
			}

			user := jwtService.GetUserFromToken(token)

			logger.Printf("Service middleware: %s", user)
			next.ServeHTTP(writer, request.WithContext(AttachUser(request.Context(), user)))
		})
	}
}

func EchoServiceAuth(
	config *io.Config,
	jwtService *slurperjwt.JWTService,
	logger *slog.Logger,
) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			var err error
			var token *jwt.Token

			adminUserContext := &contexts.AdminUserContext{
				Context: ctx,
				User:    "",
			}

			if config.AuthenticationScheme == authscheme.NONE {
				return next(adminUserContext)
			}

			logger.Debug("Starting parse of JWT token")

			sToken := echoTokenFromHeader(ctx)
			if sToken == "" {
				logger.Error("No bearer and token in authorization header")
				return ctx.String(http.StatusForbidden, "Unauthorized")
			}

			if token, err = jwtService.Parse(sToken, config.AuthSecret); err != nil {
				logger.With("error", err).Error("Error parsing JWT token in service authorization middleware")
				return ctx.String(http.StatusForbidden, "Error parsing token")
			}

			if err = jwtService.IsTokenValid(token); err != nil {
				logger.With("error", err).Error("Invalid token")
				return ctx.String(http.StatusForbidden, "Invalid token")
			}

			adminUserContext.User = jwtService.GetUserFromToken(token)
			logger.With("user", adminUserContext.User).Debug("Service middleware")

			return next(adminUserContext)
		}
	}
}

func echoTokenFromHeader(ctx echo.Context) string {
	return strings.TrimPrefix(ctx.Request().Header.Get("Authorization"), "Bearer ")
}

func tokenFromHeader(request *http.Request) string {
	return strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
}
