package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port      string
	MongoURI  string
	Database  string
	JWTSecret string
	AppOrigin string
}

func Load() (Config, error) {
	cfg := Config{
		Port:      env("PORT", "8080"),
		MongoURI:  env("MONGODB_URI", "mongodb://localhost:27017"),
		Database:  env("MONGODB_DATABASE", "business_app"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		AppOrigin: env("APP_ORIGIN", "*"),
	}
	if len(cfg.JWTSecret) < 24 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 24 characters")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
