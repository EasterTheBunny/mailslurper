package app

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"time"

	"github.com/easterthebunny/service"
	"github.com/spf13/cobra"

	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/mailslurper"
	"github.com/mailslurper/mailslurper/v2/internal/model"
	"github.com/mailslurper/mailslurper/v2/internal/smtp"
	"github.com/mailslurper/mailslurper/v2/internal/ui"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/authfactory"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/jwt"
)

var _ service.Runnable = (*HTTPService)(nil)

type HTTPServiceConfig struct {
	Version  string
	Data     Persistance
	Config   *io.Config
	Renderer *ui.TemplateRenderer
	Logger   *slog.Logger
}

type HTTPService struct {
	// dependencies
	config  *io.Config
	handler http.Handler

	// internal state
	server *http.Server
}

func NewHTTPService(config *HTTPServiceConfig) *HTTPService {
	apiHandler := &APIRouter{
		Version: config.Version,
		Data:    config.Data,
		Config:  config.Config,
		JWTService: &jwt.JWTService{
			Config: config.Config,
		},
		Logger: slog.NewLogLogger(config.Logger.Handler(), slog.LevelDebug),
	}

	router := &Router{
		Version:  config.Version,
		Config:   config.Config,
		Renderer: config.Renderer,
		AuthFactory: &authfactory.AuthFactory{
			Config: config.Config,
		},
		Logger: slog.NewLogLogger(config.Logger.Handler(), slog.LevelDebug),
		MountPaths: map[string]http.Handler{
			"/api": apiHandler.Routes(),
		},
	}

	return &HTTPService{
		config:  config.Config,
		handler: router.Routes(),
	}
}

func (s *HTTPService) Start() error {
	tracker := service.NewConnectionTracker(time.Second)
	s.server = &http.Server{
		Addr:              s.config.Public.GetBindingAddress(),
		Handler:           s.handler,
		IdleTimeout:       2000 * time.Millisecond,
		ConnState:         tracker.ConnState,
		ReadHeaderTimeout: 500 * time.Millisecond,
	}

	if s.config.Public.CertFile != "" {
		s.server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		return s.server.ListenAndServeTLS(s.config.Public.CertFile, s.config.Public.KeyFile)
	} else {
		return s.server.ListenAndServe()
	}
}

func (s *HTTPService) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPService) Close() error {
	return s.server.Close()
}

var _ service.Runnable = (*SMTPService)(nil)

type SMTPService struct {
	config   *io.Config
	database IStorage
	logger   *slog.Logger

	// internal state
	chMail   chan *model.MailItem
	pool     *smtp.ServerPool
	listener *smtp.Listener
	chClose  chan struct{}
}

func NewSMTPService(config *io.Config, logger *slog.Logger) *SMTPService {
	return &SMTPService{
		chMail:  make(chan *model.MailItem, 1_000),
		pool:    smtp.NewServerPool(logger.With("who", "SMTP Server Pool"), config.MaxWorkers),
		chClose: make(chan struct{}),
	}
}

func (s *SMTPService) Start() error {
	/*
	 * Setup receivers (subscribers) to handle new mail items.
	 */
	receivers := []mailslurper.IMailItemReceiver{
		NewDatabaseReceiver(s.database, s.logger.With("who", "Database Receiver")),
	}

	/*
	 * Setup the SMTP listener
	 */
	smtpListener, err := smtp.NewListener(
		s.logger.With("who", "SMTP Listener"),
		s.config.SMTP,
		s.chMail,
		s.pool,
		receivers,
		smtp.NewConnectionManager(s.logger.With("who", "Connection Manager"), s.config, s.chClose, s.chMail, s.pool),
	)
	if err != nil {
		s.logger.Error("There was a problem starting the SMTP listener. Exiting...")
		cobra.CheckErr(err)
	}

	s.listener = smtpListener

	return s.listener.ListenAndServe()
}

func (s *SMTPService) Shutdown(ctx context.Context) error {
	close(s.chClose)

	return s.listener.Shutdown(ctx)
}

func (s *SMTPService) Close() error {
	close(s.chClose)

	return s.Close()
}
