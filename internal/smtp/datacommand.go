// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package smtp

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log/slog"
	"mime/quotedprintable"
	"strings"

	"github.com/adampresley/webframework/sanitizer"

	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/model"
)

// DataCommandExecutor process the Data TO command.
type DataCommandExecutor struct {
	emailValidationService mailslurper.EmailValidationProvider
	logger                 *slog.Logger
	reader                 *Reader
	writer                 *Writer
	xssService             sanitizer.IXSSServiceProvider
}

// NewDataCommandExecutor creates a new struct.
func NewDataCommandExecutor(
	logger *slog.Logger,
	reader *Reader,
	writer *Writer,
	emailValidationService mailslurper.EmailValidationProvider,
	xssService sanitizer.IXSSServiceProvider,
) *DataCommandExecutor {
	return &DataCommandExecutor{
		emailValidationService: emailValidationService,
		logger:                 logger,
		reader:                 reader,
		writer:                 writer,
		xssService:             xssService,
	}
}

// Process processes the DATA command (constant DATA). When a client sends the DATA command there are three parts to the
// transmission content. Before this data can be processed this function will tell the client how to terminate the DATA
// block. We are asking clients to terminate with "\r\n.\r\n".
//
// The first part is a set of header lines. Each header line is a header key (name), followed by a colon, followed by
// the value for that header key. For example a header key might be "Subject" with a value of "Testing Mail!".
//
// After the header section there should be two sets of carriage return/line feed characters. This signals the end of
// the header block and the start of the message body.
//
// Finally when the client sends the "\r\n.\r\n" the DATA transmission portion is complete. This function will return
// the following items.
//
// 1. Headers (MailHeader)
// 2. Body breakdown (MailBody)
// 3. error structure
func (e *DataCommandExecutor) Process(streamInput string, mailItem *model.MailItem) error {
	var err error

	commandCheck := strings.Index(strings.ToLower(streamInput), "data")
	if commandCheck < 0 {
		return errors.New("Invalid command for DATA")
	}

	e.writer.SendDataResponse()

	entireMailContents, err := e.reader.ReadDataBlock()
	if err != nil {
		return fmt.Errorf("Error in DataCommandExecutor: %w", err)
	}

	if err = mailItem.Message.BuildMessages(entireMailContents); err != nil {
		e.logger.Error(fmt.Sprintf("Problem parsing message contents: %s", err.Error()))
		e.writer.SendResponse(SMTP_ERROR_TRANSACTION_FAILED)

		return fmt.Errorf("Problem parsing message contents: %w", err)
	}

	mailItem.Subject = e.getSubjectFromPart(mailItem.Message)
	mailItem.DateSent = mailslurper.ParseDateTime(mailItem.Message.GetHeader("Date"), e.logger)
	mailItem.ContentType = mailItem.Message.GetHeader("Content-Type")
	mailItem.TransferEncoding = mailItem.Message.GetHeader("Content-Transfer-Encoding")

	if len(mailItem.Message.MessageParts) > 0 {
		for _, m := range mailItem.Message.MessageParts {
			e.recordMessagePart(m, mailItem)
		}

		if mailItem.HTMLBody != "" {
			mailItem.Body = mailItem.HTMLBody
		} else {
			mailItem.Body = mailItem.TextBody
		}

	} else {
		if mailItem.Body, err = e.getBodyContent(mailItem.Message.GetBody()); err != nil {
			e.writer.SendResponse(SMTP_ERROR_TRANSACTION_FAILED)

			return fmt.Errorf("Problem reading body: %w", err)
		}
	}

	if mailItem.Body, err = e.decodeBody(mailItem.Body, mailItem.ContentType, mailItem.TransferEncoding); err != nil {
		e.writer.SendResponse(SMTP_ERROR_TRANSACTION_FAILED)

		return fmt.Errorf("Problem decoding body: %w", err)
	}

	e.logger.Debug(fmt.Sprintf("Subject: %s", mailItem.Subject))
	e.logger.Debug(fmt.Sprintf("Date: %s", mailItem.DateSent))
	e.logger.Debug(fmt.Sprintf("Content-Type: %s", mailItem.ContentType))
	e.logger.Debug(fmt.Sprintf("Body: %s", mailItem.Body))
	e.logger.Debug(fmt.Sprintf("Transfer Encoding: %s", mailItem.TransferEncoding))

	e.writer.SendOkResponse()

	return nil
}

func (e *DataCommandExecutor) addAttachment(messagePart model.ISMTPMessagePart, mailItem *model.MailItem) error {
	headers := &model.AttachmentHeader{
		ContentType:             messagePart.GetHeader("Content-Type"),
		MIMEVersion:             messagePart.GetHeader("MIME-Version"),
		ContentTransferEncoding: messagePart.GetHeader("Content-Transfer-Encoding"),
		ContentDisposition:      messagePart.GetContentDisposition(),
		FileName:                messagePart.GetFilenameFromContentDisposition(),
		Logger:                  e.logger,
	}

	e.logger.Debug(fmt.Sprintf("Adding attachment: %v", headers))

	attachment := model.NewAttachment(headers, messagePart.GetBody(), e.xssService)

	if e.messagePartIsAttachment(messagePart) {
		mailItem.Attachments = append(mailItem.Attachments, attachment)
	} else {
		mailItem.InlineAttachments = append(mailItem.InlineAttachments, attachment)
	}

	return nil
}

func (e *DataCommandExecutor) decodeBody(body, contentType, transferEncoding string) (string, error) {
	var result []byte
	var err error

	switch transferEncoding {
	case "base64":
		if result, err = base64.StdEncoding.DecodeString(body); err != nil {
			return body, err
		}
		break
	case "quoted-printable":
		if result, err = ioutil.ReadAll(quotedprintable.NewReader(strings.NewReader(body))); err != nil {
			return body, err
		}
		break

	default:
		result = []byte(body)
		break
	}

	if strings.Contains(contentType, "text/plain") {
		result = []byte(strings.Replace(string(result), "\n", "<br />", -1))
	}

	return string(result), nil
}

func (e *DataCommandExecutor) getBodyContent(contents string) (string, error) {
	/*
	 * Split the DATA content by CRLF CRLF. The first item will be the data
	 * headers. Everything past that is body/message.
	 */
	headerBodySplit := strings.Split(contents, "\r\n\r\n")
	if len(headerBodySplit) < 2 {
		return "", errors.New("Expected DATA block to contain a header section and a body section")
	}

	return strings.Join(headerBodySplit[1:], "\r\n\r\n"), nil
}

func (e *DataCommandExecutor) getSubjectFromPart(part *model.SMTPMessagePart) string {
	result := part.GetHeader("Subject")

	if strings.Compare(result, "") == 0 {
		result = "(No Subject)"
	}

	return result
}

func (e *DataCommandExecutor) isMIMEType(messagePart model.ISMTPMessagePart, mimeType string) bool {
	return strings.HasPrefix(messagePart.GetContentType(), mimeType)
}

func (e *DataCommandExecutor) messagePartIsAttachment(messagePart model.ISMTPMessagePart) bool {
	return strings.Contains(messagePart.GetContentDisposition(), "attachment")
}

func (e *DataCommandExecutor) recordMessagePart(message model.ISMTPMessagePart, mailItem *model.MailItem) error {
	if e.isMIMEType(message, "text/plain") && mailItem.TextBody == "" && !e.messagePartIsAttachment(message) {
		mailItem.TextBody = message.GetBody()
	} else {
		if e.isMIMEType(message, "text/html") && mailItem.HTMLBody == "" && !e.messagePartIsAttachment(message) {
			mailItem.HTMLBody = message.GetBody()
		} else {
			if e.isMIMEType(message, "multipart") {
				for _, m := range message.GetMessageParts() {
					e.recordMessagePart(m, mailItem)
				}
			} else {
				e.addAttachment(message, mailItem)
			}
		}
	}

	return nil
}
