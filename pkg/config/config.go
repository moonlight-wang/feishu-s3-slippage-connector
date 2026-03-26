package config

import (
	"os"
	"strconv"
)

type Config struct {
	S3Endpoint            string
	S3Region              string
	S3Bucket              string
	S3AccessKey           string
	S3SecretKey           string
	ServerPort            int
	LarkVerificationToken string
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))

	return &Config{
		S3Endpoint:            getEnv("S3_ENDPOINT", ""),
		S3Region:              getEnv("S3_REGION", "us-east-1"),
		S3Bucket:              getEnv("S3_BUCKET", "logs"),
		S3AccessKey:           getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:           getEnv("S3_SECRET_KEY", ""),
		ServerPort:            port,
		LarkVerificationToken: getEnv("LARK_VERIFICATION_TOKEN", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
