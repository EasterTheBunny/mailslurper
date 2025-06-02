package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/requests"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
)

type MailRemover interface {
	DeleteMailsAfterDate(string) (int64, error)
}

type ParamFunc func(*http.Request, string) string

type DeleteMailParams struct {
	Prune string `form:"prune,omitempty" json:"prune,omitempty"`
}

// DeleteMail is a request to delete mail items. This expects a param containing a valid prune code.
//
// DELETE: /mail?prune={pruneCode}
func DeleteMail(
	data MailRemover,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		if err := response.ValidContextsAndMethod(request, http.MethodDelete); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		params, err := requests.APIQueryParams[DeleteMailParams](request)
		if err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		pruneCode := requests.PruneCode(params.Prune)
		if !pruneCode.IsValid() {
			err := fmt.Errorf("%w: prune code: %s", response.ErrInvalidInput, pruneCode.String())

			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		startDate := pruneCode.ConvertToDate()

		rowsDeleted, err := data.DeleteMailsAfterDate(startDate)
		if err != nil {
			err = fmt.Errorf("%w: problem deleting mails with code %s", err, pruneCode.String())

			response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

			return
		}

		logger.Printf("Deleting %d mails, code %s before %s", rowsDeleted, pruneCode.String(), startDate)
		response.RenderOrLog(writer, request, &response.TextResponse{
			HTTPStatusCode: http.StatusOK,
			Data:           []byte(strconv.Itoa(int(rowsDeleted))),
		}, logger)
	}
}

// GetPruneOptions retrieves the set of options available to users for pruning.
func GetPruneOptions(logger *log.Logger) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		logger.Print("responding with prune options")
		response.RenderOrLog(writer, request, &response.JSONResponse{
			HTTPStatusCode: http.StatusOK,
			Value:          requests.PruneOptions,
		}, logger)
	}
}
