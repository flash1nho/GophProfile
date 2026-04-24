package config

import (
	"os"

	"strconv"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	DBURL                    string
	DBMaxConns               int32
	DBMinConns               int32
	DBMaxConnLifetime        time.Duration
	DBMaxConnIdleTime        time.Duration
	DBHealthCheckPeriod      time.Duration
	DBConnectTimeout         time.Duration
	S3Endpoint               string
	S3Key                    string
	S3Secret                 string
	S3Bucket                 string
	RabbitURL                string
	OtelExporterOtlpEndpoint string
}

type Option func(*Config)

func WithDBURL(v string) Option {
	return func(c *Config) {
		c.DBURL = v
	}
}

func WithS3Endpoint(v string) Option {
	return func(c *Config) {
		c.S3Endpoint = v
	}
}

func WithS3Key(v string) Option {
	return func(c *Config) {
		c.S3Key = v
	}
}

func WithS3Secret(v string) Option {
	return func(c *Config) {
		c.S3Secret = v
	}
}

func WithS3Bucket(v string) Option {
	return func(c *Config) {
		c.S3Bucket = v
	}
}

func WithRabbitURL(v string) Option {
	return func(c *Config) {
		c.RabbitURL = v
	}
}

func WithOtelExporterOtlpEndpoint(v string) Option {
	return func(c *Config) {
		c.OtelExporterOtlpEndpoint = v
	}
}

func New(log *zap.Logger, opts ...Option) *Config {
	cfg := &Config{
		DBURL:                    os.Getenv("DB_URL"),
		DBMaxConns:               getEnvInt("DB_MAX_CONNS", 20),
		DBMinConns:               getEnvInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetime:        getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
		DBMaxConnIdleTime:        getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
		DBHealthCheckPeriod:      getEnvDuration("DB_HEALTHCHECK_PERIOD", time.Minute),
		DBConnectTimeout:         getEnvDuration("DB_CONNECT_TIMEOUT", 5*time.Second),
		S3Endpoint:               os.Getenv("S3_ENDPOINT"),
		S3Key:                    os.Getenv("S3_KEY"),
		S3Secret:                 os.Getenv("S3_SECRET"),
		S3Bucket:                 os.Getenv("S3_BUCKET"),
		RabbitURL:                os.Getenv("RABBIT_URL"),
		OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	validate(cfg, log)

	return cfg
}

func validate(c *Config, log *zap.Logger) {
	if c.S3Endpoint == "" {
		log.Fatal("S3_ENDPOINT is required")
	}
	if c.DBURL == "" {
		log.Fatal("DB_URL is required")
	}
	if c.RabbitURL == "" {
		log.Fatal("RABBIT_URL is required")
	}
}

func getEnvInt(key string, def int32) int32 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return int32(parsed)
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			return parsed
		}
	}
	return def
}
