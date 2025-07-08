package persistence

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/adampresley/webframework/sanitizer"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"

	"github.com/mailslurper/mailslurper/v2/internal/model"
)

//go:embed migrations/*
var migrations embed.FS

type Config struct {
	// Database determines the name of the database schema to use.
	Database string `mapstructure:"database"`
	// Dialect is the name of the database system to use.
	Dialect string `mapstructure:"dialect"`
	// Host is the host the database system is running on.
	Host string `mapstructure:"host"`
	// Password is the password for the database user to use for connecting to the database.
	Password string `mapstructure:"password"`
	// Port is the port the database system is running on.
	Port string `mapstructure:"port"`
	// URL is a datasource connection string. It can be used instead of the rest of the database configuration
	// options. If this `url` is set then it is prioritized, i.e. the rest of the options, if set, have no effect.
	//
	// Schema: `dialect://username:password@host:port/database`
	URL string `mapstructure:"url"`
	// User is the database user to use for connecting to the database.
	User string `mapstructure:"user"`
}

type ORM struct {
	db        *pop.Connection
	sanitizer sanitizer.IXSSServiceProvider
	logger    *slog.Logger
}

func NewORM(
	config Config,
	xss sanitizer.IXSSServiceProvider,
	logger *slog.Logger,
) (*ORM, error) {
	connectionDetails := &pop.ConnectionDetails{
		Pool:            5,
		IdlePool:        0,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 1 * time.Hour,
	}

	if len(config.URL) > 0 {
		connectionDetails.URL = config.URL
	} else {
		connectionDetails.Dialect = config.Dialect
		connectionDetails.Database = config.Database
		connectionDetails.Host = config.Host
		connectionDetails.Port = config.Port
		connectionDetails.User = config.User
		connectionDetails.Password = config.Password
	}

	DB, err := pop.NewConnection(connectionDetails)

	if err != nil {
		return nil, err
	}

	if err := DB.Open(); err != nil {
		return nil, err
	}

	return &ORM{db: DB, sanitizer: xss, logger: logger}, nil
}

// MigrateUp applies all pending up migrations to the Database
func (s *ORM) MigrateUp() error {
	migrationBox, err := pop.NewMigrationBox(migrations, s.db)
	if err != nil {
		return err
	}

	return migrationBox.Up()
}

// MigrateDown migrates the Database down by the given number of steps
func (s *ORM) MigrateDown(steps int) error {
	migrationBox, err := pop.NewMigrationBox(migrations, s.db)
	if err != nil {
		return err
	}

	return migrationBox.Down(steps)
}

// GetAttachment retrieves an attachment for a given mail item.
func (s *ORM) GetAttachment(attachmentID uuid.UUID) (*model.Attachment, error) {
	attachment := model.Attachment{}

	err := s.db.Find(&attachment, attachmentID)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	attachment.Sanitize(s.sanitizer)

	return &attachment, nil
}

// GetMailByID retrieves a single mail item and attachment by ID.
func (s *ORM) GetMailByID(id uuid.UUID) (*model.MailItem, error) {
	item := model.MailItem{}

	eagerPreloadFields := []string{
		"Attachments",
	}

	err := s.db.EagerPreload(eagerPreloadFields...).Find(&item, id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get mail item: %w", err)
	}

	item.Sanitize(s.sanitizer)

	return &item, nil
}

// GetMailMessageRawByID retrieves a single mail item and attachment by ID.
func (s *ORM) GetMailMessageRawByID(id uuid.UUID) (string, error) {
	item := model.MailItem{}

	err := s.db.Find(&item, id)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("failed to get body: %w", err)
	}

	return item.Body, nil
}

// GetMailCollection retrieves a slice of mail items starting at offset and getting length number of records.
func (s *ORM) GetMailCollection(offset, length int, mailSearch *MailSearch) ([]model.MailItem, error) {
	items := []model.MailItem{}

	err := addQuery(s.db, mailSearch).
		Paginate(offset, length).
		EagerPreload().All(&items)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get mail items: %w", err)
	}

	return nil, errors.New("unimplemented")
}

// GetMailCount returns the number of total records in the mail items table.
func (s *ORM) GetMailCount(mailSearch *MailSearch) (int, error) {
	return addQuery(s.db, mailSearch).Count(&model.MailItem{})
}

// DeleteMailsAfterDate deletes all mails after a specified date.
func (s *ORM) DeleteMailsAfterDate(startDate string) (int64, error) {
	parameters := []any{}

	if len(startDate) > 0 {
		parameters = append(parameters, startDate)
	}

	if err := s.db.RawQuery(getDeleteAttachmentsQuery(startDate), parameters...).Exec(); err != nil {
		return 0, fmt.Errorf("%w: Error deleting attachments for mails after %s", err, startDate)
	}

	if err := s.db.RawQuery(getDeleteMailQuery(startDate), parameters...).Exec(); err != nil {
		return 0, fmt.Errorf("%w: Error deleting mails after %s", err, startDate)
	}

	return 0, nil // TODO: count number of mail items deleted
}

// StoreMail writes a mail item and its attachments to the storage device. This returns the new mail ID.
func (s *ORM) StoreMail(mailItem *model.MailItem) error {
	vErr, err := s.db.ValidateAndCreate(&mailItem)
	if err != nil {
		return err
	}

	if vErr != nil && vErr.HasAny() {
		return fmt.Errorf("primary email object validation failed: %w", vErr)
	}

	s.logger.Info("New mail item written to database.")

	return nil
}
