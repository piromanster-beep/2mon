package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config — настройки приложения
type Config struct {
	MongoURI      string
	MaxBotToken   string
	MaxAPIURL     string
	AdminPassword string
	RateLimit     int
	QueueSize     int
	Port          string
}

// Load — загрузка из переменных окружения
func Load() (*Config, error) {
	cfg := &Config{
		MaxAPIURL: env("MAX_API_URL", "https://platform-api.max.ru"),
		RateLimit: envPositiveInt("RATE_LIMIT", 30),
		QueueSize: envPositiveInt("QUEUE_SIZE", 1000),
		Port:      env("PORT", "8080"),
	}

	// Обязательные поля
	cfg.MongoURI = os.Getenv("MONGO_URI")
	if cfg.MongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI не задан")
	}

	cfg.MaxBotToken = os.Getenv("MAX_BOT_TOKEN")
	if cfg.MaxBotToken == "" {
		return nil, fmt.Errorf("MAX_BOT_TOKEN не задан")
	}

	cfg.AdminPassword = os.Getenv("ADMIN_PASSWORD")
	if cfg.AdminPassword == "" {
		return nil, fmt.Errorf("ADMIN_PASSWORD не задан")
	}

	return cfg, nil
}

// env — строка из окружения с значением по умолчанию
func env(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// envInt — число из окружения с значением по умолчанию.
func envInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}

// envPositiveInt — как envInt, но ноль и отрицательные значения
// отбрасываются в пользу дефолта. Для RATE_LIMIT и QUEUE_SIZE это
// важно: rate.Limiter с limit <= 0 навсегда блокирует отправку, а
// канал нулевой ёмкости переполняется мгновенно.
func envPositiveInt(key string, defaultVal int) int {
	n := envInt(key, defaultVal)
	if n <= 0 {
		return defaultVal
	}
	return n
}
