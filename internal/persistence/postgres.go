// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package persistence

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/adampresley/webframework/sanitizer"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	slurperio "github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// PgSQLStorage implements the IStorage interface.
type PgSQLStorage struct {
	connectionInformation *slurperio.ConnectionInformation
	db                    *sqlx.DB
	logger                *slog.Logger
	xssService            sanitizer.IXSSServiceProvider
}

// NewPgSQLStorage creates a new storage object that interfaces to PostgreSQL.
func NewPgSQLStorage(connectionInformation *slurperio.ConnectionInformation, logger *slog.Logger) *PgSQLStorage {
	return &PgSQLStorage{
		connectionInformation: connectionInformation,
		xssService:            sanitizer.NewXSSService(),
		logger:                logger,
	}
}

// Connect to the database.
func (s *PgSQLStorage) Connect() error {
	db, err := sqlx.Connect("postgres", "")
	s.db = db

	return fmt.Errorf("Error connecting to %s: %w", s.connectionInformation.Filename, err)
}

// Disconnect does exactly what you think it does.
func (s *PgSQLStorage) Disconnect() {
	s.db.Close()
}

// Create creates the necessary database tables.
func (s *PgSQLStorage) Create() error {
	s.logger.Info("Creating database tables...")

	if _, err := os.Stat(s.connectionInformation.Filename); err == nil {
		if err := os.Remove(s.connectionInformation.Filename); err != nil {
			return fmt.Errorf("Error removing existing SQLite storage file %s: %w", s.connectionInformation.Filename, err)
		}
	}

	sqlStatement := `
		CREATE TABLE mailitem (
			id TEXT PRIMARY KEY,
			dateSent TEXT,
			fromAddress TEXT,
			toAddressList TEXT,
			subject TEXT,
			xmailer TEXT,
			body TEXT,
			contentType TEXT,
			boundary TEXT
		);`

	if _, err := s.db.Exec(sqlStatement); err != nil {
		return fmt.Errorf("Error executing query: %s: %w", sqlStatement, err)
	}

	sqlStatement = `
		CREATE TABLE attachment (
			id TEXT PRIMARY KEY,
			mailItemId TEXT,
			fileName TEXT,
			contentType TEXT,
			content TEXT
		);`

	if _, err := s.db.Exec(sqlStatement); err != nil {
		return fmt.Errorf("Error executing query: %s: %w", sqlStatement, err)
	}

	s.logger.Info("Created tables successfully.")

	return nil
}

// GetAttachment retrieves an attachment for a given mail item.
func (s *PgSQLStorage) GetAttachment(mailID, attachmentID string) (*model.Attachment, error) {
	var (
		fileName    string
		contentType string
		content     string
		rows        *sql.Rows
		err         error
	)

	result := &model.Attachment{}
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

	if rows, err = s.db.Query(getAttachmentSQL, attachmentID, mailID); err != nil {
		return result, fmt.Errorf("%w: Error getting attachment %s for mail %s: %s", err, attachmentID, mailID, getAttachmentSQL)
	}

	defer rows.Close()
	rows.Next()
	rows.Scan(&fileName, &contentType, &content)

	result.Headers = &model.AttachmentHeader{
		FileName:    fileName,
		ContentType: contentType,
		Logger:      s.logger,
	}

	result.MailID = mailID
	result.Contents = content

	return result, nil
}

// GetMailByID retrieves a single mail item and attachment by ID.
func (s *PgSQLStorage) GetMailByID(mailItemID string) (*model.MailItem, error) {
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

	if rows, err = s.db.Query(sqlQuery, mailItemID); err != nil {
		return result, fmt.Errorf("%w: Error getting mail %s: %s", err, mailItemID, sqlQuery)
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&dateSent, &fromAddress, &toAddressList, &subject, &xmailer, &body, &mailContentType, &boundary, &attachmentID, &fileName, &attachmentContentType)
		if err != nil {
			return result, fmt.Errorf("%w: Error scanning mail record %s in GetMailByID", err, mailItemID)
		}

		/*
		 * Only capture the mail item once. Every subsequent record is an attachment
		 */
		if result.ID == "" {
			result = &model.MailItem{
				ID:          mailItemID,
				DateSent:    dateSent,
				FromAddress: fromAddress,
				ToAddresses: strings.Split(toAddressList, "; "),
				Subject:     s.xssService.SanitizeString(subject),
				XMailer:     s.xssService.SanitizeString(xmailer),
				Body:        s.xssService.SanitizeString(body),
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
					FileName:    s.xssService.SanitizeString(fileName.String),
					ContentType: attachmentContentType.String,
					Logger:      s.logger,
				},
			}

			attachments = append(attachments, newAttachment)
		}
	}

	result.Attachments = attachments

	return result, nil
}

// GetMailMessageRawByID retrieves a single mail item and attachment by ID.
func (s *PgSQLStorage) GetMailMessageRawByID(mailItemID string) (string, error) {
	var result string

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

	if rows, err = s.db.Query(sqlQuery, mailItemID); err != nil {
		return result, fmt.Errorf("%w: Error getting mail %s: %s", err, mailItemID, sqlQuery)
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&dateSent, &fromAddress, &toAddressList, &subject, &xmailer, &body, &mailContentType, &boundary, &attachmentID, &fileName, &attachmentContentType)
		if err != nil {
			return result, fmt.Errorf("%w: Error scanning mail record %s in GetMailMessageRawByID", err, mailItemID)
		}

		result = body

		return result, nil
	}

	return result, nil
}

// GetMailCollection retrieves a slice of mail items starting at offset and getting length number of records. This query
// is MSSQL 2005 and higher compatible.
func (s *PgSQLStorage) GetMailCollection(offset, length int, mailSearch *MailSearch) ([]*model.MailItem, error) {
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

	sqlQuery := `
		SELECT
			  mailitem.id
			, mailitem.dateSent
			, mailitem.fromAddress
			, mailitem.toAddressList
			, mailitem.subject
			, mailitem.xmailer
			, mailitem.body
			, mailitem.contentType AS mailContentType
			, mailitem.boundary
			, attachment.id AS attachmentID
			, attachment.fileName
			, attachment.contentType AS attachmentContentType
		FROM mailitem
			LEFT JOIN attachment ON attachment.mailItemID=mailitem.id
		WHERE 1=1
	`

	sqlQuery, parameters = addSearchCriteria(sqlQuery, parameters, mailSearch)
	sqlQuery = addOrderBy(sqlQuery, "mailitem", mailSearch)

	sqlQuery = sqlQuery + `
		LIMIT ? OFFSET ?
	`

	parameters = append(parameters, length)
	parameters = append(parameters, offset)

	if rows, err = s.db.Query(sqlQuery, parameters...); err != nil {
		return result, fmt.Errorf("%w: Error getting mails: %s", err, sqlQuery)
	}

	defer rows.Close()

	currentMailItemID = ""

	for rows.Next() {
		err = rows.Scan(&mailItemID, &dateSent, &fromAddress, &toAddressList, &subject, &xmailer, &body, &mailContentType, &boundary, &attachmentID, &fileName, &attachmentContentType)
		if err != nil {
			return result, fmt.Errorf("%w: Error scanning mail record in GetMailCollection", err)
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
				ToAddresses: strings.Split(toAddressList, "; "),
				Subject:     s.xssService.SanitizeString(subject),
				XMailer:     s.xssService.SanitizeString(xmailer),
				Body:        s.xssService.SanitizeString(body),
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
					FileName:    s.xssService.SanitizeString(fileName.String),
					ContentType: attachmentContentType.String,
					Logger:      s.logger,
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

// GetMailCount returns the number of total records in the mail items table.
func (s *PgSQLStorage) GetMailCount(mailSearch *MailSearch) (int, error) {
	var mailItemCount int
	var err error

	sqlQuery, parameters := getMailCountQuery(mailSearch)
	if err = s.db.QueryRow(sqlQuery, parameters...).Scan(&mailItemCount); err != nil {
		return 0, fmt.Errorf("%w: Error getting mail count: %s", err, sqlQuery)
	}

	return mailItemCount, nil
}

// DeleteMailsAfterDate deletes all mails after a specified date.
func (s *PgSQLStorage) DeleteMailsAfterDate(startDate string) (int64, error) {
	sqlQuery := ""
	parameters := []any{}
	var result sql.Result
	var rowsAffected int64
	var err error

	if len(startDate) > 0 {
		parameters = append(parameters, startDate)
	}

	sqlQuery = getDeleteAttachmentsQuery(startDate)
	if _, err = s.db.Exec(sqlQuery, parameters...); err != nil {
		return 0, fmt.Errorf("%w: Error deleting attachments for mails after %s: %s", err, startDate, sqlQuery)
	}

	sqlQuery = getDeleteMailQuery(startDate)
	if result, err = s.db.Exec(sqlQuery, parameters...); err != nil {
		return 0, fmt.Errorf("%w: Error deleting mails after %s: %s", err, startDate, sqlQuery)
	}

	if rowsAffected, err = result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("%w: Error getting count of rows affected when deleting mails", err)
	}

	return rowsAffected, err
}

// StoreMail writes a mail item and its attachments to the storage device. This returns the new mail ID.
func (s *PgSQLStorage) StoreMail(mailItem *model.MailItem) (string, error) {
	var err error
	var transaction *sql.Tx
	var statement *sql.Stmt

	/*
	 * Create a transaction and insert the new mail item
	 */
	if transaction, err = s.db.Begin(); err != nil {
		return "", fmt.Errorf("%w: Error starting transaction in StoreMail", err)
	}

	/*
	 * Insert the mail item
	 */
	if statement, err = transaction.Prepare(getInsertMailQuery()); err != nil {
		return "", fmt.Errorf("%w: Error preparing insert statement in StoreMail", err)
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

		return "", fmt.Errorf("%w: Error inserting new mail item in StoreMail", err)
	}

	statement.Close()

	/*
	 * Insert attachments
	 */
	if err = storeAttachments(mailItem.ID, transaction, mailItem.Attachments); err != nil {
		transaction.Rollback()

		return "", fmt.Errorf("%w: Error storing attachments to mail %s", err, mailItem.ID)
	}

	transaction.Commit()
	s.logger.Info("New mail item written to database.")

	return mailItem.ID, nil
}
