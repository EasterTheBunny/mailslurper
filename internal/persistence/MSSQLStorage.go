// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package persistence

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/adampresley/webframework/sanitizer"
	_ "github.com/denisenkom/go-mssqldb" // MSSQL
	"github.com/pkg/errors"

	slurperio "github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

/*
MSSQLStorage implements the IStorage interface
*/
type MSSQLStorage struct {
	connectionInformation *slurperio.ConnectionInformation
	db                    *sql.DB
	logger                *slog.Logger
	xssService            sanitizer.IXSSServiceProvider
}

/*
NewMSSQLStorage creates a new storage object that interfaces to MSSQL
*/
func NewMSSQLStorage(connectionInformation *slurperio.ConnectionInformation, logger *slog.Logger) *MSSQLStorage {
	return &MSSQLStorage{
		connectionInformation: connectionInformation,
		xssService:            sanitizer.NewXSSService(),
		logger:                logger,
	}
}

/*
Connect to the database
*/
func (storage *MSSQLStorage) Connect() error {
	connectionString := fmt.Sprintf("Server=%s;Port=%d;User Id=%s;Password=%s;Database=%s",
		storage.connectionInformation.Address,
		storage.connectionInformation.Port,
		storage.connectionInformation.UserName,
		storage.connectionInformation.Password,
		storage.connectionInformation.Database,
	)

	db, err := sql.Open("mssql", connectionString)

	storage.db = db
	return err
}

/*
Disconnect does exactly what you think it does
*/
func (storage *MSSQLStorage) Disconnect() {
	storage.db.Close()
}

/*
Create does nothing for MSSQL
*/
func (storage *MSSQLStorage) Create() error {
	return nil
}

/*
GetAttachment retrieves an attachment for a given mail item
*/
func (storage *MSSQLStorage) GetAttachment(mailID, attachmentID string) (*model.Attachment, error) {
	result := &model.Attachment{}
	var err error
	var rows *sql.Rows

	var fileName string
	var contentType string
	var content string

	getAttachmentSQL := `
		SELECT
			  attachment.fileName
			, attachment.contentType
			, attachment.content
		FROM attachment
		WHERE
			id=?
			AND mailItemId=?
	`

	if rows, err = storage.db.Query(getAttachmentSQL, attachmentID, mailID); err != nil {
		return result, errors.Wrapf(err, "Error getting attachment %s for mail %s: %s", attachmentID, mailID, getAttachmentSQL)
	}

	defer rows.Close()
	rows.Next()
	rows.Scan(&fileName, &contentType, &content)

	result.Headers = &model.AttachmentHeader{
		FileName:    fileName,
		ContentType: contentType,
		Logger:      storage.logger,
	}

	result.MailID = mailID
	result.Contents = content
	return result, nil
}

/*
GetMailByID retrieves a single mail item and attachment by ID
*/
func (storage *MSSQLStorage) GetMailByID(mailItemID string) (*model.MailItem, error) {
	result := &model.MailItem{}
	attachments := make([]*model.Attachment, 0, 5)

	var err error
	var rows *sql.Rows

	var dateSent string
	var fromAddress string
	var toAddressList string
	var subject string
	var xmailer string
	var body string
	var boundary sql.NullString
	var attachmentID sql.NullString
	var fileName sql.NullString
	var mailContentType string
	var attachmentContentType sql.NullString

	sqlQuery := getMailAndAttachmentsQuery(" AND mailitem.id=? ")

	if rows, err = storage.db.Query(sqlQuery, mailItemID); err != nil {
		return result, errors.Wrapf(err, "Error getting mail %s: %s", mailItemID, sqlQuery)
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&dateSent, &fromAddress, &toAddressList, &subject, &xmailer, &body, &mailContentType, &boundary, &attachmentID, &fileName, &attachmentContentType)
		if err != nil {
			return result, errors.Wrapf(err, "Error scanning mail record %s in GetMailByID", mailItemID)
		}

		/*
		 * Only capture the mail item once. Every subsequent record is an attachment
		 */
		if result.ID == "" {
			result = &model.MailItem{
				ID:          mailItemID,
				DateSent:    dateSent,
				FromAddress: fromAddress,
				ToAddresses: model.NewMailAddressCollectionFromStringList(toAddressList),
				Subject:     storage.xssService.SanitizeString(subject),
				XMailer:     storage.xssService.SanitizeString(xmailer),
				Body:        storage.xssService.SanitizeString(body),
				ContentType: mailContentType,
			}

			if boundary.Valid {
				result.Boundary = boundary.String
			}
		}

		if attachmentID.Valid {
			newAttachment := &model.Attachment{
				ID:     attachmentID.String,
				MailID: mailItemID,
				Headers: &model.AttachmentHeader{
					FileName:    storage.xssService.SanitizeString(fileName.String),
					ContentType: attachmentContentType.String,
					Logger:      storage.logger,
				},
			}

			attachments = append(attachments, newAttachment)
		}
	}

	result.Attachments = attachments
	return result, nil
}

/*
GetMailMessageRawByID retrieves a single mail item and attachment by ID
*/
func (storage *MSSQLStorage) GetMailMessageRawByID(mailItemID string) (string, error) {
	result := ""

	var err error
	var rows *sql.Rows

	var dateSent string
	var fromAddress string
	var toAddressList string
	var subject string
	var xmailer string
	var body string
	var boundary sql.NullString
	var attachmentID sql.NullString
	var fileName sql.NullString
	var mailContentType string
	var attachmentContentType sql.NullString

	sqlQuery := getMailAndAttachmentsQuery(" AND mailitem.id=? ")

	if rows, err = storage.db.Query(sqlQuery, mailItemID); err != nil {
		return result, errors.Wrapf(err, "Error getting mail %s: %s", mailItemID, sqlQuery)
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&dateSent, &fromAddress, &toAddressList, &subject, &xmailer, &body, &mailContentType, &boundary, &attachmentID, &fileName, &attachmentContentType)
		if err != nil {
			return result, errors.Wrapf(err, "Error scanning mail record %s in GetMailByID", mailItemID)
		}
		result = body
		return result, nil
	}
	return result, nil
}

/*
GetMailCollection retrieves a slice of mail items starting at offset and getting length number
of records. This query is MSSQL 2005 and higher compatible.
*/
func (storage *MSSQLStorage) GetMailCollection(offset, length int, mailSearch *MailSearch) ([]*model.MailItem, error) {
	result := make([]*model.MailItem, 0, 50)
	attachments := make([]*model.Attachment, 0, 5)

	var err error
	var rows *sql.Rows

	var currentMailItemID string
	var currentMailItem *model.MailItem
	var parameters []interface{}

	var mailItemID string
	var dateSent string
	var fromAddress string
	var toAddressList string
	var subject string
	var xmailer string
	var body string
	var mailContentType string
	var boundary sql.NullString
	var attachmentID sql.NullString
	var fileName sql.NullString
	var attachmentContentType sql.NullString

	/*
	 * This query is MSSQL 2005 and higher compatible
	 */
	sqlQuery := `
		WITH pagedMailItems AS (
			SELECT
				  mailitem.id
				, mailitem.dateSent
				, mailitem.fromAddress
				, mailitem.toAddressList
				, mailitem.subject
				, mailitem.xmailer
				, mailitem.body
				, mailitem.contentType
				, mailitem.boundary
				, ROW_NUMBER() OVER (ORDER BY mailitem.dateSent DESC) AS rowNumber
			FROM mailitem
			WHERE 1=1
	`

	sqlQuery, parameters = addSearchCriteria(sqlQuery, parameters, mailSearch)

	sqlQuery = sqlQuery + `
		)
		SELECT
			  pagedMailItems.id AS mailItemID
			, pagedMailItems.dateSent
			, pagedMailItems.fromAddress
			, pagedMailItems.toAddressList
			, pagedMailItems.subject
			, pagedMailItems.xmailer
			, pagedMailItems.body
			, pagedMailItems.contentType AS mailContentType
			, pagedMailItems.boundary
			, attachment.id AS attachmentID
			, attachment.fileName
			, attachment.contentType AS attachmentContentType

		FROM pagedMailItems
			LEFT JOIN attachment ON attachment.mailItemID=pagedMailItems.id

		WHERE
			pagedMailItems.rowNumber BETWEEN ? AND ?

	`

	sqlQuery = addOrderBy(sqlQuery, "pagedMailItems", mailSearch)

	parameters = append(parameters, offset)
	parameters = append(parameters, offset+length)

	if rows, err = storage.db.Query(sqlQuery, parameters...); err != nil {
		return result, errors.Wrapf(err, "Error getting mails: %s", sqlQuery)
	}

	defer rows.Close()

	currentMailItemID = ""

	for rows.Next() {
		err = rows.Scan(&mailItemID, &dateSent, &fromAddress, &toAddressList, &subject, &xmailer, &body, &mailContentType, &boundary, &attachmentID, &fileName, &attachmentContentType)
		if err != nil {
			return result, errors.Wrapf(err, "Error scanning mail record in GetMailCollection")
		}

		if currentMailItemID != mailItemID {
			/*
			 * If we have a mail item we are working with place the attachments with it.
			 * Then reset everything in prep for the next mail item and batch of attachments
			 */
			if currentMailItemID != "" {
				currentMailItem.Attachments = attachments
				result = append(result, currentMailItem)
			}

			currentMailItem = &model.MailItem{
				ID:          mailItemID,
				DateSent:    dateSent,
				FromAddress: fromAddress,
				ToAddresses: model.NewMailAddressCollectionFromStringList(toAddressList),
				Subject:     storage.xssService.SanitizeString(subject),
				XMailer:     storage.xssService.SanitizeString(xmailer),
				Body:        storage.xssService.SanitizeString(body),
				ContentType: mailContentType,
			}

			if boundary.Valid {
				currentMailItem.Boundary = boundary.String
			}

			currentMailItemID = mailItemID
			attachments = make([]*model.Attachment, 0, 5)
		}

		if attachmentID.Valid {
			newAttachment := &model.Attachment{
				ID:     attachmentID.String,
				MailID: mailItemID,
				Headers: &model.AttachmentHeader{
					FileName:    storage.xssService.SanitizeString(fileName.String),
					ContentType: attachmentContentType.String,
					Logger:      storage.logger,
				},
			}

			attachments = append(attachments, newAttachment)
		}
	}

	/*
	 * Attach our straggler
	 */
	if currentMailItemID != "" {
		currentMailItem.Attachments = attachments
		result = append(result, currentMailItem)
	}

	return result, nil
}

/*
GetMailCount returns the number of total records in the mail items table
*/
func (storage *MSSQLStorage) GetMailCount(mailSearch *MailSearch) (int, error) {
	var mailItemCount int
	var err error

	sqlQuery, parameters := getMailCountQuery(mailSearch)
	if err = storage.db.QueryRow(sqlQuery, parameters...).Scan(&mailItemCount); err != nil {
		return 0, errors.Wrapf(err, "Error getting mail count: %s", sqlQuery)
	}

	return mailItemCount, nil
}

/*
DeleteMailsAfterDate deletes all mails after a specified date
*/
func (storage *MSSQLStorage) DeleteMailsAfterDate(startDate string) (int64, error) {
	sqlQuery := ""
	parameters := []interface{}{}
	var result sql.Result
	var rowsAffected int64
	var err error

	if len(startDate) > 0 {
		parameters = append(parameters, startDate)
	}

	sqlQuery = getDeleteAttachmentsQuery(startDate)
	if _, err = storage.db.Exec(sqlQuery, parameters...); err != nil {
		return 0, errors.Wrapf(err, "Error deleting attachments for mails after %s: %s", startDate, sqlQuery)
	}

	sqlQuery = getDeleteMailQuery(startDate)
	if result, err = storage.db.Exec(sqlQuery, parameters...); err != nil {
		return 0, errors.Wrapf(err, "Error deleting mails after %s: %s", startDate, sqlQuery)
	}

	if rowsAffected, err = result.RowsAffected(); err != nil {
		return 0, errors.Wrapf(err, "Error getting count of rows affected when deleting mails")
	}

	return rowsAffected, err
}

/*
StoreMail writes a mail item and its attachments to the storage device. This returns the new mail ID
*/
func (storage *MSSQLStorage) StoreMail(mailItem *model.MailItem) (string, error) {
	var err error
	var transaction *sql.Tx
	var statement *sql.Stmt

	/*
	 * Create a transaction and insert the new mail item
	 */
	if transaction, err = storage.db.Begin(); err != nil {
		return "", errors.Wrapf(err, "Error starting transaction in StoreMail")
	}

	/*
	 * Insert the mail item
	 */
	if statement, err = transaction.Prepare(getInsertMailQuery()); err != nil {
		return "", errors.Wrapf(err, "Error preparing insert statement in StoreMail")
	}

	_, err = statement.Exec(
		mailItem.ID,
		mailItem.DateSent,
		mailItem.FromAddress,
		strings.Join(mailItem.ToAddresses, "; "),
		mailItem.Subject,
		mailItem.XMailer,
		mailItem.Body,
		mailItem.ContentType,
		mailItem.Boundary,
	)

	if err != nil {
		transaction.Rollback()
		return "", errors.Wrapf(err, "Error inserting new mail item in StoreMail")
	}

	statement.Close()

	/*
	 * Insert attachments
	 */
	if err = storeAttachments(mailItem.ID, transaction, mailItem.Attachments); err != nil {
		transaction.Rollback()
		return "", errors.Wrapf(err, "Error storing attachments to mail %s", mailItem.ID)
	}

	transaction.Commit()
	storage.logger.Info("New mail item written to database.")

	return mailItem.ID, nil
}
