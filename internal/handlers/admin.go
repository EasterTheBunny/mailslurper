package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/middleware"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/requests"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
	slurperio "github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/ui"
)

func GetServiceSettings(
	version string,
	config *slurperio.Config,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		user := middleware.GetUser(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, user); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		response.RenderOrLog(writer, request, &response.JSONResponse{
			HTTPStatusCode: http.StatusOK,
			Value: &response.ServiceSettings{
				AuthenticationScheme: config.AuthenticationScheme,
				URL:                  config.Public.GetURL(),
				Version:              version,
			},
		}, logger)
	}
}

func GetVersion(
	version string,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		user := middleware.GetUser(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, user); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		response.RenderOrLog(writer, request, &response.JSONResponse{
			HTTPStatusCode: http.StatusOK,
			Value: &response.Version{
				Version: version,
			},
		}, logger)
	}
}

// Deprecated:
// GetVersionFromMaster returns the same version as GetVersion.
func GetVersionFromMaster(
	version string,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		user := middleware.GetUser(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, user); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		response.RenderOrLog(writer, request, &response.JSONResponse{
			HTTPStatusCode: http.StatusOK,
			Value: &response.Version{
				Version: version,
			},
		}, logger)
	}
}

func ApplyTheme(
	config *slurperio.Config,
	renderer *ui.TemplateRenderer,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		user := middleware.GetUser(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodPost, user); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		body, err := io.ReadAll(request.Body)
		if err != nil {
			err = fmt.Errorf("%w: failed to read request body", err)

			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		var requestedTheme requests.ApplyThemeRequest
		if err := json.Unmarshal(body, &requestedTheme); err != nil {
			err = fmt.Errorf("%w: failed to read request body", err)

			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		config.Theme = requestedTheme.Theme

		// TODO: the apply theme function should either be converted to a user setting in a cookie or removed
		// the global config should not be writable from the UI
		if err = config.WriterFunc(); err != nil {
			err = fmt.Errorf("%w: Error saving configuration file in ApplyTheme", err)

			response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

			return
		}

		renderer.LoadTemplates()
		response.RenderOrLog(writer, request, &response.TextResponse{
			HTTPStatusCode: http.StatusOK,
			Data:           []byte(""),
		}, logger)
	}
}
