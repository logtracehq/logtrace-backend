package config

import (
	"time"

	"gitlab.com/logtrace/logtrace"
)

// ENUM(prod,dev)
type LogMode string

// ENUM(production,development)
type EnvType string

type GoogleAuth struct {
	Code string `yaml:"code" mapstructure:"code"`
}

type Otel struct {
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	UseTLS   bool   `yaml:"use_tls" mapstructure:"use_tls"`
	Headers  string `yaml:"headers" mapstructure:"headers"`
}

type HTTP struct {
	Port      int `yaml:"port" mapstructure:"port"`
	RateLimit struct {
		Type              string        `yaml:"type" mapstructure:"type"`
		IsEnabled         bool          `yaml:"is_enabled" mapstructure:"is_enabled"`
		RequestsPerMinute int           `yaml:"requests_per_minute" mapstructure:"requests_per_minute"`
		BurstInterval     time.Duration `yaml:"burst_interval" mapstructure:"burst_interval"`
	} `yaml:"rate_limit" mapstructure:"rate_limit"`
	Swagger struct {
		Port int `yaml:"port" mapstructure:"port"`
	} `yaml:"swagger" mapstructure:"swagger"`
	Metrics struct {
		Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
		Username string `yaml:"username" mapstructure:"username"`
		Password string `yaml:"password" mapstructure:"password"`
	} `yaml:"metrics" mapstructure:"metrics"`
}

type Database struct {
	Redis struct {
		DSN string `yaml:"dsn" mapstructure:"dsn"`
	} `yaml:"redis" mapstructure:"redis"`
	Postgres struct {
		DSN          string        `yaml:"dsn" mapstructure:"dsn"`
		LogQueries   bool          `yaml:"log_queries" mapstructure:"log_queries"`
		QueryTimeout time.Duration `yaml:"query_timeout" mapstructure:"query_timeout"`
	} `yaml:"postgres" mapstructure:"postgres"`
}

type JWT struct {
	Key        string `yaml:"key" mapstructure:"key"`
	Audience   string `yaml:"audience" mapstructure:"audience"`
	PrivateKey string `yaml:"private_key" mapstructure:"private_key"`
	PublicKey  string `yaml:"public_key" mapstructure:"public_key"`
}

type Auth struct {
	Google struct {
		ClientID     string   `yaml:"client_id" mapstructure:"client_id"`
		ClientSecret string   `yaml:"client_secret" mapstructure:"client_secret"`
		RedirectURL  string   `yaml:"redirect_url" mapstructure:"redirect_url"`
		Scopes       []string `yaml:"scopes" mapstructure:"scopes"`
	} `yaml:"google" mapstructure:"google"`
	JWT JWT `yaml:"jwt" mapstructure:"jwt"`
}

type Frontend struct {
	AppURL string `yaml:"app_url" mapstructure:"app_url"`
}

type APIKey struct {
	HashSecret string `yaml:"hash_secret" mapstructure:"hash_secret"`
}

type Email struct {
	Provider   string         `yaml:"provider" mapstructure:"provider"`
	Sender     logtrace.Email `yaml:"sender" mapstructure:"sender"`
	SenderName string         `yaml:"sender_name" mapstructure:"sender_name"`
	Resend     struct {
		APIKey        string `yaml:"api_key" mapstructure:"api_key"`
		WebhookSecret string `yaml:"webhook_secret" mapstructure:"webhook_secret"`
	} `yaml:"resend" mapstructure:"resend"`
}

type Billing struct {
	TrialDays int `yaml:"trial_days" mapstructure:"trial_days"`
}

type Uploader struct {
	S3 struct {
		Region                 string `yaml:"region" mapstructure:"region"`
		AccessKey              string `yaml:"access_key" mapstructure:"access_key"`
		AccessSecret           string `yaml:"access_secret" mapstructure:"access_secret"`
		Endpoint               string `yaml:"endpoint" mapstructure:"endpoint"`
		UseTLS                 bool   `yaml:"use_tls" mapstructure:"use_tls"`
		LogOperations          bool   `yaml:"log_operations" mapstructure:"log_operations"`
		Bucket                 string `yaml:"bucket" mapstructure:"bucket"`
		CloudflareBucketDomain string `yaml:"cloudflare_bucket_domain" mapstructure:"cloudflare_bucket_domain"`
	} `yaml:"s3" mapstructure:"s3"`
}

type Config struct {
	DBHost     string     `yaml:"db_host" mapstructure:"db_host"`
	DBPort     string     `yaml:"db_port" mapstructure:"db_port"`
	DBUser     string     `yaml:"db_user" mapstructure:"db_user"`
	DBPassword string     `yaml:"db_password" mapstructure:"db_password"`
	DBName     string     `yaml:"db_name" mapstructure:"db_name"`
	Env        EnvType    `yaml:"env" mapstructure:"env"`
	TZ         string     `yaml:"tz" mapstructure:"tz"`
	DBSSLMode  string     `yaml:"db_ssl_mode" mapstructure:"db_ssl_mode"`
	HTTP       HTTP       `yaml:"http" mapstructure:"http"`
	GoogleAuth GoogleAuth `yaml:"google_auth" mapstructure:"google_auth"`
	Port       string     `yaml:"port" mapstructure:"port"`
	LogLevel   string     `yaml:"log_level" mapstructure:"log_level"`
	Logging    struct {
		Mode LogMode `yaml:"mode" mapstructure:"mode" json:"mode"`
	} `yaml:"logging" mapstructure:"logging" json:"logging"`
	Otel       Otel     `yaml:"otel" mapstructure:"otel"`
	Auth       Auth     `yaml:"auth" mapstructure:"auth"`
	Database   Database `yaml:"database" mapstructure:"database"`
	Frontend   Frontend `yaml:"frontend" mapstructure:"frontend"`
	Email      Email    `yaml:"email" mapstructure:"email"`
	APIKey     APIKey   `yaml:"api_key" mapstructure:"api_key"`
	Billing    Billing  `yaml:"billing" mapstructure:"billing"`
	Uploader   Uploader `yaml:"uploader" mapstructure:"uploader"`
	CSRFSecret string   `yaml:"csrf_secret" mapstructure:"csrf_secret"`
}
