package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("MAX_BOT_TOKEN", "token")
	t.Setenv("ADMIN_PASSWORD", "pass")
	t.Setenv("MAX_API_URL", "")
	t.Setenv("RATE_LIMIT", "")
	t.Setenv("QUEUE_SIZE", "")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Документация (README.md, deploy/HTTPS.md) указывает platform-api.max.ru,
	// а не устаревший api.max.ru — дефолт должен совпадать.
	if want := "https://platform-api.max.ru"; cfg.MaxAPIURL != want {
		t.Errorf("MaxAPIURL = %q, want %q", cfg.MaxAPIURL, want)
	}
	if cfg.RateLimit != 30 {
		t.Errorf("RateLimit = %d, want 30", cfg.RateLimit)
	}
	if cfg.QueueSize != 1000 {
		t.Errorf("QueueSize = %d, want 1000", cfg.QueueSize)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
}

func TestLoadRequiresCredentials(t *testing.T) {
	t.Setenv("MONGO_URI", "")
	t.Setenv("MAX_BOT_TOKEN", "")
	t.Setenv("ADMIN_PASSWORD", "")

	if _, err := Load(); err == nil {
		t.Fatal("Load() = nil error, want error when required env vars are empty")
	}
}
