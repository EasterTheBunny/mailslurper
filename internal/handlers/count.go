package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
	"github.com/mailslurper/mailslurper/v2/internal/persistence"
)

type MailCounter interface {
	GetMailCount(*persistence.MailSearch) (int, error)
}

// GetMailCount returns the number of mail items in storage.
func GetMailCount(
	data MailCounter,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		mailItemCount, err := data.GetMailCount(&persistence.MailSearch{})
		if err != nil {
			err = fmt.Errorf("%w: problem getting mail item count", err)

			response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

			return
		}

		logger.Printf("Mail item count - %d", mailItemCount)

		response.RenderOrLog(writer, request, &response.JSONResponse{
			HTTPStatusCode: http.StatusOK,
			Value: &response.MailCountResponse{
				MailCount: mailItemCount,
			},
		}, logger)
	}
}
