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

// Ноль и отрицательные значения для RATE_LIMIT/QUEUE_SIZE должны
// отбрасываться в пользу дефолта, иначе лимитер блокирует отправку
// навсегда (limit <= 0), а очередь переполняется мгновенно (cap 0).
func TestLoadClampsNonPositiveLimits(t *testing.T) {
	t.Setenv("MONGO_URI", "mongodb://localhost:27017")
	t.Setenv("MAX_BOT_TOKEN", "token")
	t.Setenv("ADMIN_PASSWORD", "pass")

	for _, tc := range []struct{ rate, queue string }{
		{"0", "0"},
		{"-5", "-1"},
		{"abc", ""},
	} {
		t.Setenv("RATE_LIMIT", tc.rate)
		t.Setenv("QUEUE_SIZE", tc.queue)

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load(RATE_LIMIT=%q, QUEUE_SIZE=%q) error: %v", tc.rate, tc.queue, err)
		}
		if cfg.RateLimit != 30 {
			t.Errorf("RATE_LIMIT=%q: RateLimit = %d, want 30", tc.rate, cfg.RateLimit)
		}
		if cfg.QueueSize != 1000 {
			t.Errorf("QUEUE_SIZE=%q: QueueSize = %d, want 1000", tc.queue, cfg.QueueSize)
		}
	}
}
