package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	DBSSLMode       string
	EmbeddingAPIURL string
	EmbeddingModel  string
	RerankerAPIURL  string
	RerankerModel   string
	LLMChatAPIURL   string
	LLMChatModel    string
}

func Load() (*Config, error) {
	// .env 파일 로드
	_ = godotenv.Load(".env.local")

	return &Config{
		DBHost:          os.Getenv("DB_HOST"),
		DBPort:          os.Getenv("DB_PORT"),
		DBUser:          os.Getenv("DB_USER"),
		DBPassword:      os.Getenv("DB_PASSWORD"),
		DBName:          os.Getenv("DB_NAME"),
		DBSSLMode:       os.Getenv("DB_SSLMODE"),
		EmbeddingAPIURL: os.Getenv("EMBEDDING_API_URL"),
		EmbeddingModel:  os.Getenv("EMBEDDING_MODEL"),
		RerankerAPIURL:  os.Getenv("RERANKER_API_URL"),
		RerankerModel:   os.Getenv("RERANKER_MODEL"),
		LLMChatAPIURL:   os.Getenv("LLMCHAT_API_URL"),
		LLMChatModel:    os.Getenv("LLMCHAT_MODEL"),
	}, nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
		c.DBSSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
