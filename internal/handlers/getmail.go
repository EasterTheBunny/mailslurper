package handlers

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/middleware"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/requests"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
	"github.com/mailslurper/mailslurper/v2/internal/model"
	"github.com/mailslurper/mailslurper/v2/internal/persistence"
)

type MailCollectionGetter interface {
	MailCounter
	GetMailCollection(int, int, *persistence.MailSearch) ([]model.MailItem, error)
}

type MailMessageRawGetter interface {
	GetMailMessageRawByID(uuid.UUID) (string, error)
}

type GetMailCollectionParams struct {
	PageNumber string `form:"pageNumber,omitempty" json:"pageNumber,omitempty"`
	Message    string `form:"message,omitempty" json:"message,omitempty"`
	Start      string `form:"start,omitempty" json:"start,omitempty"`
	End        string `form:"end,omitempty" json:"end,omitempty"`
	From       string `form:"from,omitempty" json:"from,omitempty"`
	To         string `form:"to,omitempty" json:"to,omitempty"`

	OrderByField     string `form:"orderby,omitempty" json:"orderby,omitempty"`
	OrderByDirection string `form:"dir,omitempty" json:"dir,omitempty"`
}

// GetMailCollection returns a collection of mail items. This is constrianed by a page number. A page of data contains
// 50 items.
//
// GET: /mails?pageNumber={pageNumber}
func GetMailCollection(
	data MailCollectionGetter,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		params, err := requests.APIQueryParams[GetMailCollectionParams](request)
		if err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		var pageNumber int
		var mailCollection []model.MailItem
		var totalRecordCount int

		/*
		 * Validate incoming arguments. A page is currently 50 items, hard coded
		 */
		if params.PageNumber == "" {
			pageNumber = 1
		} else {
			if pageNumber, err = strconv.Atoi(params.PageNumber); err != nil {
				err = fmt.Errorf("%w: Invalid page number passed to GetMailCollection - %s", err, params.PageNumber)

				response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

				return
			}
		}

		length := 50
		offset := (pageNumber - 1) * length

		/*
		 * Retrieve mail items
		 */
		mailSearch := &persistence.MailSearch{
			Message: params.Message,
			Start:   params.Start,
			End:     params.End,
			From:    params.From,
			To:      params.To,

			OrderByField:     params.OrderByField,
			OrderByDirection: params.OrderByDirection,
		}

		if mailCollection, err = data.GetMailCollection(offset, length, mailSearch); err != nil {
			err = fmt.Errorf("%w: problem getting mail collection", err)

			response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

			return
		}

		if totalRecordCount, err = data.GetMailCount(mailSearch); err != nil {
			err = fmt.Errorf("%w: problem getting record count in GetMailCollection", err)

			response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

			return
		}

		totalPages := int(math.Ceil(float64(totalRecordCount / length)))
		if totalPages*length < totalRecordCount {
			totalPages++
		}

		logger.Printf("Mail collection page %d retrieved", pageNumber)

		response.RenderOrLog(writer, request, &response.JSONResponse{
			HTTPStatusCode: http.StatusOK,
			Value: &response.MailCollectionResponse{
				MailItems:    mailCollection,
				TotalPages:   totalPages,
				TotalRecords: totalRecordCount,
			},
		}, logger)
	}
}

// GetMail returns a single mail item by ID.
//
// GET: /mail/{mailId}
func GetMail(
	data MailCollectionGetter,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		mailItem := middleware.GetMailItem(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, mailItem); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		mailBody, convertSucess := convertFromBase64(mailItem.Body)
		if convertSucess == true {
			mailItem.Body = mailBody
			mailItem.HTMLBody = mailBody
		}

		logger.Printf("Mail item %s retrieved", mailItem.ID)
		response.RenderOrLog(writer, request, &response.JSONResponse{
			HTTPStatusCode: http.StatusOK,
			Value:          mailItem,
		}, logger)
	}
}

// GetMailMessage returns the message contents of a single mail item
//
// GET: /mail/{mailId}/message
func GetMailMessage(
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		mailItem := middleware.GetMailItem(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, mailItem); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		mailBody, convertSucess := convertFromBase64(mailItem.Body)
		if convertSucess == true {
			response.RenderOrLog(writer, request, &response.HTMLResponse{
				HTTPStatusCode: http.StatusOK,
				Value:          mailBody,
			}, logger)

			return
		}

		logger.Printf("Mail item %s retrieved", mailItem.ID)
		response.RenderOrLog(writer, request, &response.HTMLResponse{
			HTTPStatusCode: http.StatusOK,
			Value:          mailItem.Body,
		}, logger)
	}
}

// GetMailMessageRaw returns the message contents of a single mail item
//
// GET: /mail/{mailId}/messageraw
func GetMailMessageRaw(
	data MailMessageRawGetter,
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		mailItem := middleware.GetMailItem(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, mailItem); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		/*
		* Retrieve the mail item
		 */
		body, err := data.GetMailMessageRawByID(mailItem.ID)
		if err != nil {
			err = fmt.Errorf("%w: Problem getting mail item %s in GetMailMessageRaw", err, mailItem.ID)

			response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

			return
		}

		logger.Printf("Mail item %s retrieved", mailItem.ID)
		response.RenderOrLog(writer, request, &response.HTMLResponse{
			HTTPStatusCode: http.StatusOK,
			Value:          body,
		}, logger)
	}
}

// DownloadAttachment retrieves binary database from storage and streams it back to the caller.
//
// GET: /mail/{mailID}/attachment/{attachmentID}
func DownloadAttachment(
	logger *log.Logger,
) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		attachment := middleware.GetMailItemAttachment(request.Context())
		if err := response.ValidContextsAndMethod(request, http.MethodGet, attachment); err != nil {
			response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

			return
		}

		var (
			data []byte
			err  error
		)

		/*
		 * Decode the base64 data and stream it back
		 */
		if attachment.IsContentBase64() {
			if data, err = base64.StdEncoding.DecodeString(attachment.Contents); err != nil {
				err = fmt.Errorf("%w: Problem decoding attachment %s", err, attachment.ID)

				response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

				return
			}
		} else {
			data = []byte(attachment.Contents)
		}

		logger.Printf("Attachment %s retrieved", attachment.ID)
		response.RenderOrLog(writer, request, &response.DataResponse{
			HTTPStatusCode: http.StatusOK,
			Data:           data,
		}, logger)
	}
}

func convertFromBase64(s string) (string, bool) {
	var mailBody []byte
	var err error

	if mailBody, err = base64.StdEncoding.DecodeString(strings.Replace(s, " ", "", -1)); err == nil {
		return string(mailBody[:]), true
	}

	return "", false
}
