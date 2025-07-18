package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid"

	"github.com/mailslurper/mailslurper/v2/internal/handlers/requests"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

type MailGetter interface {
	GetMailByID(uuid.UUID) (*model.MailItem, error)
}

// MailCtx ...
func MailCtx(
	data MailGetter,
	pFn func(*http.Request, string) string,
	logger *log.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			mailID, err := uuid.FromString(pFn(request, requests.MailIDPathParam))
			if err != nil {
				err := fmt.Errorf("%w: mail id", response.ErrNotFound)

				response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

				return
			}

			// retrieve the mail item
			item, err := data.GetMailByID(mailID)
			if err != nil {
				err = fmt.Errorf("%w: Problem getting mail item %s", err, mailID)

				response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

				return
			}

			ctx := AttachMailItem(request.Context(), *item)

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

type MailAttachmentGetter interface {
	GetAttachment(uuid.UUID, uuid.UUID) (*model.Attachment, error)
}

// MailAttachmentCtx ...
func MailAttachmentCtx(
	data MailAttachmentGetter,
	pFn func(*http.Request, string) string,
	logger *log.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			mailItem := GetMailItem(request.Context())
			if err := response.ValidContextsAndMethod(request, http.MethodGet, mailItem); err != nil {
				response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

				return
			}

			attachmentID, err := uuid.FromString(pFn(request, requests.MailAttachmentIDPathParam))
			if err != nil {
				err := fmt.Errorf("%w: attachment id", response.ErrNotFound)

				response.RenderOrLog(writer, request, response.HTTPBadRequest(err), logger)

				return
			}

			// retrieve the mail item attachment
			item, err := data.GetAttachment(mailItem.ID, attachmentID)
			if err != nil {
				err = fmt.Errorf("%w: Problem getting mail item attachment %s", err, attachmentID)

				response.RenderOrLog(writer, request, response.HTTPInternalServerError(err), logger)

				return
			}

			ctx := AttachMailAttachment(request.Context(), *item)

			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}
