// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.
package controllers

import (
	"bytes"
	"encoding/base64"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/mailslurper/mailslurper/v2/internal/app"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/requests"
	"github.com/mailslurper/mailslurper/v2/internal/handlers/response"
	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/model"
	"github.com/mailslurper/mailslurper/v2/internal/persistence"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/auth"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/authfactory"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/jwt"
	"github.com/mailslurper/mailslurper/v2/pkg/contexts"
)

/*
ServiceController provides methods for handling service endpoints.
This is to primarily support the API
*/
type ServiceController struct {
	AuthFactory   authfactory.IAuthFactory
	Config        *io.Config
	Database      app.IStorage
	JWTService    jwt.IJWTService
	Logger        *slog.Logger
	ServerVersion string
}

/*
DeleteMail is a request to delete mail items. This expects a body containing
a DeleteMailRequest object.

	DELETE: /mail/{pruneCode}
*/
func (c *ServiceController) DeleteMail(ctx echo.Context) error {
	var err error
	var deleteMailRequest *mailslurper.DeleteMailRequest
	var rowsDeleted int64

	context := contexts.GetAdminContext(ctx)

	if err = ctx.Bind(&deleteMailRequest); err != nil {
		c.Logger.Error("Invalid delete request in DeleteMail: %s", err.Error())
		return context.String(http.StatusBadRequest, "Invalid delete request")
	}

	if !deleteMailRequest.PruneCode.IsValid() {
		c.Logger.Error("Attempt to use invalid prune code - %s", deleteMailRequest.PruneCode)
		return context.String(http.StatusBadRequest, "Invalid prune type")
	}

	startDate := deleteMailRequest.PruneCode.ConvertToDate()

	if rowsDeleted, err = c.Database.DeleteMailsAfterDate(startDate); err != nil {
		c.Logger.Error("Problem deleting mails with code %s - %s", deleteMailRequest.PruneCode.String(), err.Error())
		return context.String(http.StatusInternalServerError, "There was a problem deleting mails")
	}

	c.Logger.Info("Deleting %d mails, code %s before %s", rowsDeleted, deleteMailRequest.PruneCode.String(), startDate)
	return context.String(http.StatusOK, strconv.Itoa(int(rowsDeleted)))
}

func (c *ServiceController) Head(ctx echo.Context) error {
	c.Logger.Info("Just HEAD")
	return ctx.NoContent(http.StatusOK)
}

/*
GetMail returns a single mail item by ID.

	GET: /mail/{id}
*/
func (c *ServiceController) GetMail(ctx echo.Context) error {
	var mailID string
	var result *model.MailItem
	var err error
	var mailBody string
	var convertSucess bool

	context := contexts.GetAdminContext(ctx)

	mailID = context.Param("id")

	/*
	 * Retrieve the mail item
	 */
	if result, err = c.Database.GetMailByID(mailID); err != nil {
		c.Logger.Error("Problem getting mail item %s - %s", mailID, err.Error())
		return context.String(http.StatusInternalServerError, "Problem getting mail item")
	}
	if mailBody, convertSucess = c.ConvertFromBase64(result.Body); convertSucess == true {
		result.Body = mailBody
		result.HTMLBody = mailBody
	}

	c.Logger.Info("Mail item %s retrieved", mailID)
	return context.JSON(http.StatusOK, result)
}

/*
GetMailCollection returns a collection of mail items. This is constrianed
by a page number. A page of data contains 50 items.

	GET: /mails?pageNumber={pageNumber}
*/
func (c *ServiceController) GetMailCollection(ctx echo.Context) error {
	var err error
	var pageNumberString string
	var pageNumber int
	var mailCollection []*model.MailItem
	var totalRecordCount int

	context := contexts.GetAdminContext(ctx)

	/*
	 * Validate incoming arguments. A page is currently 50 items, hard coded
	 */
	pageNumberString = context.QueryParam("pageNumber")
	if pageNumberString == "" {
		pageNumber = 1
	} else {
		if pageNumber, err = strconv.Atoi(pageNumberString); err != nil {
			c.Logger.Error("Invalid page number passed to GetMailCollection - %s", pageNumberString)
			return context.String(http.StatusBadRequest, "A valid page number is required")
		}
	}

	length := 50
	offset := (pageNumber - 1) * length

	/*
	 * Retrieve mail items
	 */
	mailSearch := &persistence.MailSearch{
		Message: context.QueryParam("message"),
		Start:   context.QueryParam("start"),
		End:     context.QueryParam("end"),
		From:    context.QueryParam("from"),
		To:      context.QueryParam("to"),

		OrderByField:     context.QueryParam("orderby"),
		OrderByDirection: context.QueryParam("dir"),
	}

	if mailCollection, err = c.Database.GetMailCollection(offset, length, mailSearch); err != nil {
		c.Logger.Error("Problem getting mail collection - %s", err.Error())
		return context.String(http.StatusInternalServerError, "Problem getting mail collection")
	}

	if totalRecordCount, err = c.Database.GetMailCount(mailSearch); err != nil {
		c.Logger.Error("Problem getting record count in GetMailCollection - %s", err.Error())
		return context.String(http.StatusInternalServerError, "Error getting record count")
	}

	totalPages := int(math.Ceil(float64(totalRecordCount / length)))
	if totalPages*length < totalRecordCount {
		totalPages++
	}

	c.Logger.Info("Mail collection page %d retrieved", pageNumber)

	result := &response.MailCollectionResponse{
		MailItems:    mailCollection,
		TotalPages:   totalPages,
		TotalRecords: totalRecordCount,
	}

	return context.JSON(http.StatusOK, result)
}

/*
GetMailCount returns the number of mail items in storage.

	GET: /mailcount
*/
func (c *ServiceController) GetMailCount(ctx echo.Context) error {
	var err error
	var mailItemCount int

	context := contexts.GetAdminContext(ctx)

	/*
	 * Get the count
	 */
	if mailItemCount, err = c.Database.GetMailCount(&persistence.MailSearch{}); err != nil {
		c.Logger.Error("Problem getting mail item count in GetMailCount - %s", err.Error())
		return context.String(http.StatusInternalServerError, "Problem getting mail count")
	}

	c.Logger.Info("Mail item count - %d", mailItemCount)

	result := &response.MailCountResponse{
		MailCount: mailItemCount,
	}

	return context.JSON(http.StatusOK, result)
}

/*
GetMailMessage returns the message contents of a single mail item

	GET: /mail/{id}/message
*/
func (c *ServiceController) GetMailMessage(ctx echo.Context) error {
	var mailID string
	var mailItem *model.MailItem
	var err error
	var mailBody string
	var convertSucess bool

	context := contexts.GetAdminContext(ctx)

	mailID = context.Param("id")

	/*
	 * Retrieve the mail item
	 */
	if mailItem, err = c.Database.GetMailByID(mailID); err != nil {
		c.Logger.Error("Problem getting mail item %s in GetMailMessage - %s", mailID, err.Error())
		return context.String(http.StatusInternalServerError, "Problem getting mail item")
	}
	if mailBody, convertSucess = c.ConvertFromBase64(mailItem.Body); convertSucess == true {
		return context.HTML(http.StatusOK, mailBody)
	}

	c.Logger.Info("Mail item %s retrieved", mailID)
	return context.HTML(http.StatusOK, mailItem.Body)
}

func (c *ServiceController) ConvertFromBase64(s string) (string, bool) {
	var mailBody []byte
	var err error

	if mailBody, err = base64.StdEncoding.DecodeString(strings.Replace(s, " ", "", -1)); err == nil {
		return string(mailBody[:]), true
	}
	return "", false

}

/*
GetMailMessageRaw returns the message contents of a single mail item

	GET: /mail/{id}/messageraw
*/
func (c *ServiceController) GetMailMessageRaw(ctx echo.Context) error {
	var mailID string
	var body string
	var err error

	context := contexts.GetAdminContext(ctx)

	mailID = context.Param("id")

	/*
	 * Retrieve the mail item
	 */
	if body, err = c.Database.GetMailMessageRawByID(mailID); err != nil {
		c.Logger.Error("Problem getting mail item %s in GetMailMessageRaw - %s", mailID, err.Error())
		return context.String(http.StatusInternalServerError, "Problem getting mail item")
	}

	c.Logger.Info("Mail item %s retrieved", mailID)
	return context.HTML(http.StatusOK, body)
}

/*
GetPruneOptions retrieves the set of options available to users for pruning

	GET: /pruneoptions
*/
func (c *ServiceController) GetPruneOptions(ctx echo.Context) error {
	context := contexts.GetAdminContext(ctx)
	return context.JSON(http.StatusOK, requests.PruneOptions)
}

/*
DownloadAttachment retrieves binary database from storage and streams
it back to the caller

	GET: /mail/{mailID}/attachment/{attachmentID}
*/
func (c *ServiceController) DownloadAttachment(ctx echo.Context) error {
	var err error
	var attachmentID string
	var mailID string

	var attachment *model.Attachment
	var data []byte

	context := contexts.GetAdminContext(ctx)
	mailID = context.Param("mailID")
	attachmentID = context.Param("attachmentID")

	/*
	 * Retrieve the attachment
	 */
	if attachment, err = c.Database.GetAttachment(mailID, attachmentID); err != nil {
		c.Logger.Error("Problem getting attachment %s - %s", attachmentID, err.Error())
		return context.String(http.StatusInternalServerError, "Error getting attachment")
	}

	/*
	 * Decode the base64 data and stream it back
	 */
	if attachment.IsContentBase64() {
		if data, err = base64.StdEncoding.DecodeString(attachment.Contents); err != nil {
			c.Logger.Error("Problem decoding attachment %s - %s", attachmentID, err.Error())
			return context.String(http.StatusInternalServerError, "Cannot decode attachment")
		}
	} else {
		data = []byte(attachment.Contents)
	}

	c.Logger.Info("Attachment %s retrieved", attachmentID)

	reader := bytes.NewReader(data)
	return context.Stream(http.StatusOK, attachment.Headers.ContentType, reader)
}

/*
Login is an endpoint used to create a JWT token for use in service calls.
This also stores that token in an in-memory cache so when a user logs
out that token can be rendered invalid
*/
func (c *ServiceController) Login(ctx echo.Context) error {
	var err error
	var token string
	var encryptedToken string

	authService := c.AuthFactory.Get()
	credentials := &auth.AuthCredentials{
		UserName: ctx.FormValue("userName"),
		Password: ctx.FormValue("password"),
	}

	if err = authService.Login(credentials); err != nil {
		c.Logger.With("error", err).Error("Invalid service login attempt")
		return ctx.String(http.StatusForbidden, "Invalid credentials")
	}

	if token, err = c.JWTService.CreateToken(c.Config.AuthSecret, credentials.UserName); err != nil {
		c.Logger.With("error", err).Error("Problem creating token in service login")
		return ctx.String(http.StatusInternalServerError, "Problem creating JWT token")
	}

	if encryptedToken, err = c.JWTService.EncryptToken(token); err != nil {
		c.Logger.With("error", err).Error("Error encrypting JWT token")
		return ctx.String(http.StatusInternalServerError, "Error encrypting JWT token")
	}

	c.Logger.With("token", encryptedToken).Debug("Encrypted JWT token generated")
	return ctx.String(http.StatusOK, encryptedToken)
}

func (c *ServiceController) Logout(ctx echo.Context) error {
	return contexts.GetAdminContext(ctx).String(http.StatusOK, "OK")
}

func (c *ServiceController) Version(ctx echo.Context) error {
	return contexts.GetAdminContext(ctx).String(http.StatusOK, c.ServerVersion)
}
