package cmd

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/adampresley/webframework/sanitizer"
	"github.com/easterthebunny/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mailslurper/mailslurper/v2/internal/app"
	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/persistence"
)

func init() {
	smtpCmd.Flags().IntVarP(&config.SMTP.Port, "port", "p", 8080, "port for http service to listen on")
	smtpCmd.Flags().StringVarP(&config.SMTP.Address, "listen", "l", "127.0.0.1", "ip address to listen on")
}

var (
	smtpCmd = &cobra.Command{
		Use:   "smtp",
		Short: "Run the smtp mailslurper service.",
		Long:  `Run the smtp mailslurper service.`,
		Run: func(cmd *cobra.Command, _ []string) {
			vpr := viper.New()

			// bind flags before reading the config
			bindFlags(vpr, cmd)
			setConfigDefaults(vpr)
			readConfig(configPath, "yaml", "", vpr)
			cobra.CheckErr(config.Validate())

			logLevel := io.LevelError
			if verbose {
				logLevel = io.LevelDebug
			}

			logger := io.NewLogger(cmd.OutOrStdout(), io.LogFormat(logFormat), logLevel)
			xss := sanitizer.NewXSSService()

			logger.Debug("Starting MailSlurper SMTP Service", "version", "v"+cmd.Version)
			orm, err := persistence.NewORM(config.Database, xss, logger)
			cobra.CheckErr(err)

			mgr := service.NewRecoverableServiceManager(
				service.WithRecoverWait(5*time.Second),
				service.WithLogger(logger),
				service.RecoverOnError,
			)

			cobra.CheckErr(mgr.Add(app.NewSMTPService(&config, xss, orm, logger)))

			ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			context.AfterFunc(ctx, func() {
				_ = mgr.Close()
			})

			_ = mgr.Start()
		},
	}
)
