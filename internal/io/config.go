// Copyright 2013-2018 Adam Presley. All rights reserved
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package io

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mailslurper/mailslurper/v2/internal/persistence"
	"github.com/mailslurper/mailslurper/v2/pkg/auth/authscheme"
)

var (
	ErrInvalidAdminAddress    = errors.New("Invalid administrator address: admin.address")
	ErrInvalidPublicAddress   = errors.New("Invalid service address: public.address")
	ErrInvalidSMTPAddress     = errors.New("Invalid SMTP address: smtp.address")
	ErrInvalidDatabaseDialect = errors.New("Invalid database dialiect. Valid values are 'sqlite', 'mysql', 'mssql', `postgres`")
	ErrInvalidDatabaseHost    = errors.New("Invalid database host: database.host")
	ErrInvalidDatabaseName    = errors.New("Invalid database name: database.name")
	ErrKeyFileNotFound        = errors.New("Key file not found")
	ErrCertFileNotFound       = errors.New("Certificate file not found")
	ErrNeedCertPair           = errors.New("Please provide both a key file and a cert file")
	ErrInvalidAuthScheme      = errors.New("Invalid authentication scheme. Valid values are 'basic': authenticationScheme")
	ErrMissingAuthSecret      = errors.New("Missing authentication secret. An authentication secret is requried when authentication is enabled: authSecret")
	ErrMissingAuthSalt        = errors.New("Missing authentication salt. A salt value is required when authentication is enabled: authSalt")
	ErrNoUsersConfigured      = errors.New("No users configured. When authentication is enabled you must have at least 1 valid user: credentials")

	defaultNixConfigPath     = filepath.Base("~/.config/mailslurper")
	defaultWindowsConfigPath = filepath.Base(`%appdata%\mailslurper`)
	defaultConfigName        = "config"
)

// Config contains settings for how to bind servers and connect to databases.
type Config struct {
	Public     ListenConfig       `mapstructure:"public"`
	SMTP       ListenConfig       `mapstructure:"smtp"`
	Database   persistence.Config `mapstructure:"database"`
	MaxWorkers int                `mapstructure:"maxWorkers"`
	Theme      string             `mapstructure:"theme"`

	AuthSecret           string            `mapstructure:"authSecret"`
	AuthSalt             string            `mapstructure:"authSalt"`
	AuthenticationScheme string            `mapstructure:"authenticationScheme"`
	AuthTimeoutInMinutes int               `mapstructure:"authTimeoutInMinutes"`
	Credentials          map[string]string `mapstructure:"credentials"`

	// WriterFunc allows the config to be persisted.
	WriterFunc func() error `mapstructure:"-"`
}

func (c *Config) GetTheme() string {
	if c.Theme == "" {
		return "default"
	}

	return c.Theme
}

type ListenConfig struct {
	Address   string `mapstructure:"address"`
	Port      int    `mapstructure:"port"`
	PublicURL string `mapstructure:"publicURL"`
	CertFile  string `mapstructure:"certificateFile"`
	KeyFile   string `mapstructure:"keyFile"`
}

func (c ListenConfig) Validate() error {
	if (c.KeyFile == "" && c.CertFile != "") || (c.KeyFile != "" && c.CertFile == "") {
		return ErrNeedCertPair
	}

	if c.KeyFile != "" && c.CertFile != "" {
		if !c.isValidFile(c.KeyFile) {
			return ErrKeyFileNotFound
		}

		if !c.isValidFile(c.CertFile) {
			return ErrCertFileNotFound
		}
	}

	return nil
}

// IsSSL returns true if cert files are provided for the SMTP server and the services tier.
func (c ListenConfig) IsSSL() bool {
	return c.KeyFile != "" && c.CertFile != ""
}

func (c *ListenConfig) GetBindingAddress() string {
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

func (c *ListenConfig) GetURL() string {
	if c.PublicURL != "" {
		return c.PublicURL
	}

	result := "http"

	if c.CertFile != "" && c.KeyFile != "" {
		result += "s"
	}

	return result + fmt.Sprintf("://%s:%d", c.Address, c.Port)
}

func (c ListenConfig) isValidFile(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func (config Config) Validate() error {
	if config.Public.Address == "" {
		return ErrInvalidPublicAddress
	}

	if config.SMTP.Address == "" {
		return ErrInvalidSMTPAddress
	}

	if err := config.Public.Validate(); err != nil {
		return err
	}

	if config.AuthenticationScheme != "" {
		if !authscheme.IsValidAuthScheme(config.AuthenticationScheme) {
			return ErrInvalidAuthScheme
		}

		if config.AuthSecret == "" {
			return ErrMissingAuthSecret
		}

		if config.AuthSalt == "" {
			return ErrMissingAuthSalt
		}

		if len(config.Credentials) < 1 {
			return ErrNoUsersConfigured
		}
	}

	return nil
}
