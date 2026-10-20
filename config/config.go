package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	JWTSecret     string
	PostgresDSN   string
	MongoURI      string
	MongoDatabase string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	return Config{
		Port:      port,
		JWTSecret: secret,
		PostgresDSN: fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			env("POSTGRES_USER", "postgres"),
			env("POSTGRES_PASSWORD", "postgres"),
			env("POSTGRES_HOST", "localhost"),
			env("POSTGRES_PORT", "5432"),
			env("POSTGRES_DB", "playerook_db")),
		MongoURI:      env("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: env("MONGO_DB", "playerook_chat_db"),
	}, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
