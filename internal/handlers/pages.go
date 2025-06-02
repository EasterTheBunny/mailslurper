package handlers

import (
	"log"
	"net/http"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/middleware"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/ui"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/auth"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/authfactory"
)

func PageIndex(
	config *io.Config,
	renderer *ui.TemplateRenderer,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		user := middleware.GetUser(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, user); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		data := mailslurper.Page{
			PublicWWWURL: config.Public.GetURL(),
			Theme:        config.GetTheme(),
			Title:        "Mail",
			User:         *user,
		}

		// TODO: write to buffer instead of directly to response
		if err := renderer.Render(writer, "mainLayout:index", data, nil); err != nil {
			logger.Print(err.Error())
		}
	}
}

func PageAdmin(
	config *io.Config,
	renderer *ui.TemplateRenderer,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		user := middleware.GetUser(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, user); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		data := mailslurper.Page{
			PublicWWWURL: config.Public.GetURL(),
			Theme:        config.GetTheme(),
			Title:        "Admin",
			User:         *user,
		}

		// TODO: write to buffer instead of directly to response
		if err := renderer.Render(writer, "mainLayout:admin", data, nil); err != nil {
			logger.Print(err.Error())
		}
	}
}

func PageManageSavedSearches(
	config *io.Config,
	renderer *ui.TemplateRenderer,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		user := middleware.GetUser(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, user); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		data := mailslurper.Page{
			PublicWWWURL: config.Public.GetURL(),
			Theme:        config.GetTheme(),
			Title:        "Manage Saved Searches",
			User:         *user,
		}

		// TODO: write to buffer instead of directly to response
		if err := renderer.Render(writer, "mainLayout:manageSavedSearches", data, nil); err != nil {
			logger.Print(err.Error())
		}
	}
}

// PageLogin renders the login page.
func PageLogin(
	config *io.Config,
	renderer *ui.TemplateRenderer,
	pFn func(*http.Request, string) string,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		data := mailslurper.Page{
			PublicWWWURL: config.Public.GetURL(),
			Theme:        config.GetTheme(),
		}

		message := pFn(request, "message")
		if message != "" {
			data.Message = message
			data.Error = true
		}

		// TODO: write to buffer instead of directly to response
		if err := renderer.Render(writer, "mainLayout:login", data, nil); err != nil {
			logger.Print(err.Error())
		}
	}
}

// Deprecated:
// ActionPerformLogin validates the provided username and password and returns a redirect.
func ActionPerformLogin(
	factory *authfactory.AuthFactory,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		if err := response.ValidContextsAndMethod(request, http.MethodPost); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		_ = request.ParseForm()

		authProvider := factory.Get()
		credentials := &auth.AuthCredentials{
			UserName: request.FormValue("userName"),
			Password: request.FormValue("password"),
		}

		// TODO: this login function is redundant on user validation. remove this once the
		// full conversion to stateless pages is complete.
		if err := authProvider.Login(credentials); err != nil {
			logger.Printf("%s: Admin authentication error", err)
			http.Redirect(writer, request, "/login?message=Invalid user name or password", http.StatusNotFound)

			return
		}

		http.Redirect(writer, request, "/", http.StatusFound)
	}
}

// Deprecated:
// ActionLogout returns a redirect and nothing else.
func ActionLogout() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "/login", http.StatusFound)
	}
}
