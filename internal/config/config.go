package config

import (
	"os"

	"go.uber.org/zap"
)

type Config struct {
	DBURL      string
	S3Endpoint string
	S3Key      string
	S3Secret   string
	S3Bucket   string
	RabbitURL  string
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

func New(log *zap.Logger, opts ...Option) *Config {
	cfg := &Config{
		DBURL:      os.Getenv("DB_URL"),
		S3Endpoint: os.Getenv("S3_ENDPOINT"),
		S3Key:      os.Getenv("S3_KEY"),
		S3Secret:   os.Getenv("S3_SECRET"),
		S3Bucket:   os.Getenv("S3_BUCKET"),
		RabbitURL:  os.Getenv("RABBIT_URL"),
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
