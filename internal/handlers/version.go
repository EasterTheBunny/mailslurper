package handlers

import (
	"log"
	"net/http"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
)

func Version(
	version string,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		logger.Print("returning version", version)
		response.RenderOrLog(writer, request, &response.TextResponse{
			HTTPStatusCode: http.StatusOK,
			Data:           []byte(version),
		}, logger)
	}
}
