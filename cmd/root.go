package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mailslurper/mailslurper/v2/internal/app"
	"github.com/mailslurper/mailslurper/v2/internal/io"
)

func init() {
	rootCmd.AddCommand(allCmd)
	rootCmd.AddCommand(smtpCmd)
	rootCmd.AddCommand(httpCmd)

	// runtime options
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Absolute location of the config.json. Default reads the config from the users home config directory.")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "Format for logging. 'text' or 'json'. Anything else will result in discarded logs.")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "log extra information for debug perposes")
	rootCmd.PersistentFlags().BoolVar(&autoStartBrowser, "auto-start-browser", false, "open browser on startup")

	// config file overrides
	rootCmd.PersistentFlags().StringVar(&config.Public.Address, "public.address", "127.0.0.1", "Address for public web service to listen on.")
	rootCmd.PersistentFlags().IntVar(&config.Public.Port, "public.port", 8080, "Port for public web service to listen on.")
}

var (
	configPath, logFormat     string
	verbose, autoStartBrowser bool
	config                    io.Config

	// TODO: do not use global variables for dependencies
	database app.IStorage

	rootCmd = &cobra.Command{
		Short: "Run the mailslurper service.",
		Long:  `Run the mailslurper service.`,
		Run:   func(cmd *cobra.Command, _ []string) {},
	}
)

func Execute(version string) {
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setConfigDefaults(vpr *viper.Viper) {
	config.WriterFunc = vpr.WriteConfig
	_ = vpr.SafeWriteConfig()
}

// readConfig configures the command to read from a defined file path or default to a file named by the default name
// in the home directory. Viper is also set to automatically read from environment variables.
func readConfig(path, defaultType, defaultName string, vpr *viper.Viper) {
	if path != "" {
		// Use config file from the flag.
		vpr.SetConfigFile(path)
	} else {
		// Find home directory.
		home, err := os.UserConfigDir()
		cobra.CheckErr(err)

		home = filepath.Join(home, "mailslurper")
		cobra.CheckErr(os.MkdirAll(home, 0755))

		// Search config in home directory with name ".cobra" (without extension).
		vpr.AddConfigPath(home)
		vpr.SetConfigType(defaultType)
		vpr.SetConfigName(defaultName)
	}

	vpr.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	vpr.AutomaticEnv()

	if err := vpr.ReadInConfig(); err == nil {
		log.Println("Using config file:", vpr.ConfigFileUsed())
		// logger.DebugContext(cmd.Context(), "Using configuration from config", "file", config)
	}

	cobra.CheckErr(vpr.Unmarshal(&config))
}

func bindFlags(vpr *viper.Viper, cmd *cobra.Command) {
	vpr.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
	vpr.BindPFlag("log-format", cmd.Flags().Lookup("log-format"))
	vpr.BindPFlag("public.address", cmd.Flags().Lookup("public.address"))
	vpr.BindPFlag("public.port", cmd.Flags().Lookup("public.port"))
}

func setupDatabase(logger *slog.Logger) {
	var err error

	/*
	 * Setup global database connection handle
	 */
	storageType, databaseConnection := config.GetDatabaseConfiguration()

	if database, err = app.ConnectToStorage(storageType, databaseConnection, logger); err != nil {
		logger.Error("Error connecting to storage type with a connection string", "type", int(storageType), "connection", databaseConnection.String())
		cobra.CheckErr(err)
	}
}
