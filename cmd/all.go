package cmd

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/easterthebunny/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mailslurper/mailslurper/v2/internal/app"
	"github.com/mailslurper/mailslurper/v2/internal/io"
	"github.com/mailslurper/mailslurper/v2/internal/ui"
)

var (
	allCmd = &cobra.Command{
		Use:   "all",
		Short: "Run the complete mailslurper service.",
		Long:  `Run the complete mailslurper service.`,
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

			renderer := ui.NewTemplateRenderer()
			logger := io.NewLogger(cmd.OutOrStdout(), io.LogFormat(logFormat), logLevel)

			logger.Debug("Starting MailSlurper Server", "version", "v"+cmd.Version)
			setupDatabase(logger)

			mgr := service.NewRecoverableServiceManager(
				service.WithRecoverWait(5*time.Second),
				service.WithLogger(logger),
				service.RecoverOnError,
			)

			appConfig := &app.HTTPServiceConfig{
				Version:  cmd.Version,
				Data:     database,
				Config:   &config,
				Renderer: renderer,
				Logger:   logger,
			}
			cobra.CheckErr(mgr.Add(app.NewHTTPService(appConfig)))
			cobra.CheckErr(mgr.Add(app.NewSMTPService(&config, logger)))

			ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			context.AfterFunc(ctx, func() {
				_ = mgr.Close()
			})

			if autoStartBrowser {
				ui.StartBrowser(&config, logger)
			}

			_ = mgr.Start()
		},
	}
)
