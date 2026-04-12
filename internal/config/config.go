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

func Load(log *zap.Logger) *Config {
	cfg := &Config{
		DBURL:      os.Getenv("DB_URL"),
		S3Endpoint: os.Getenv("S3_ENDPOINT"),
		S3Key:      os.Getenv("S3_KEY"),
		S3Secret:   os.Getenv("S3_SECRET"),
		S3Bucket:   os.Getenv("S3_BUCKET"),
		RabbitURL:  os.Getenv("RABBIT_URL"),
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
