package config

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gitlab.com/logbase/logbase/internal/pkg/util"
)

var (
	// Version describes the version of the current build.
	Version = "dev"

	// Commit describes the commit of the current build.
	Commit = "none"

	// Date describes the date of the current build.
	Date = time.Now().UTC()
)

const (
	defaultConfigFilePath = "config"
	envPrefix             = "LOGBASE_"
)

func (googleCfg GoogleAuth) Validate(ctx context.Context, code string) error {
	if googleCfg.Code == "" {
		return fmt.Errorf("google auth code is required")
	}

	return nil
}

func (cfg *Config) SetDefaultValues() {
	viper.SetDefault("http.port", "8080")
	viper.SetDefault("http.log_level", "info")
	viper.SetDefault("database.postgres.dsn", "postgres://postgres:password@localhost:5432/logbase?sslmode=disable")
	viper.SetDefault("database.redis.dsn", "redis://localhost:6379")
	viper.SetDefault("email.resend.api_key", "resend")
	viper.SetDefault("email.resend.webhook_secret", "secret")
	viper.SetDefault("http.metrics.username", "username")
	viper.SetDefault("http.metrics.password", "password")
	viper.SetDefault("auth.jwt.key", "thisisaweaksecret")
	viper.SetDefault("auth.jwt.audience", "logbase")
	viper.SetDefault("api_key.hash_secret", "hash_secret")
}

func (c *Config) Validate() error {
	if _, err := mail.ParseAddress(c.Email.Sender.String()); err != nil {
		return errors.New("email sender is invalid")
	}

	if c.HTTP.Port < 0 {
		return errors.New("please provide a valid HTTP port number greater than 0")
	}

	if c.HTTP.Port == 0 {
		c.HTTP.Port = 8080
	}
	if c.HTTP.Swagger.Port == 0 {
		c.HTTP.Swagger.Port = 8081
	}

	if util.IsStringEmpty(c.HTTP.Metrics.Password) {
		return errors.New("metrics password must be provided if metrics is enabled")
	}

	if util.IsStringEmpty(c.HTTP.Metrics.Username) {
		return errors.New("metrics username must be provided if metrics is enabled")
	}

	if util.IsStringEmpty(c.APIKey.HashSecret) {
		return errors.New("you must provide a hash secret for your api keys")
	}

	if util.IsStringEmpty(c.Auth.Google.ClientID) {
		return errors.New("google client secret is needed")
	}
	if util.IsStringEmpty(c.Auth.Google.ClientSecret) {
		return errors.New("client secret is required")
	}

	if util.IsStringEmpty(c.Auth.JWT.Key) {
		return errors.New("please provide your JWT key")
	}

	// trial days is required. We do not want to
	// somehow enforce users to provide a payment method/card
	// before they can get into the app
	if c.Billing.TrialDays < 0 {
		return errors.New("trial days must be 0 or greater than 0")
	}

	return nil
}

func InitializeConfig(cfg *Config, pathToFile string) error {
	homePath, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	viper.AddConfigPath(filepath.Join(homePath, ".config", defaultConfigFilePath))
	viper.AddConfigPath(pathToFile)
	viper.AddConfigPath(".")

	viper.SetConfigName(defaultConfigFilePath)
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	cfg.SetDefaultValues()

	bindEnvs(viper.GetViper(), "", cfg)

	return viper.Unmarshal(cfg)
}

func bindEnvs(v *viper.Viper, prefix string, iface any) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	if ifv.Kind() == reflect.Ptr {
		ifv = ifv.Elem()
		ift = ift.Elem()
	}

	for i := 0; i < ift.NumField(); i++ {
		fieldv := ifv.Field(i)
		t := ift.Field(i)
		name := t.Name
		tag, ok := t.Tag.Lookup("mapstructure")
		if ok {
			name = tag
		}

		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		switch fieldv.Kind() {
		case reflect.Struct:
			bindEnvs(v, path, fieldv.Addr().Interface())
		default:
			envKey := strings.ToUpper(strings.ReplaceAll(path, ".", "_"))
			if err := v.BindEnv(path, envPrefix+envKey); err != nil {
				panic(err)
			}
		}
	}
}
