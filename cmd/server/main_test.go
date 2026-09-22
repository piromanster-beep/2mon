package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Регрессия: команды бота нужно регистрировать через PATCH /me/commands.
// Раньше код слал PATCH /me — MAX такой запрос команд не принимает,
// и подсказки команд в клиенте не появлялись.
func TestRegisterBotCommandsUsesMeCommands(t *testing.T) {
	var gotPath, gotMethod, gotAuth, gotCT string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := registerBotCommands(srv.URL, "bot-token"); err != nil {
		t.Fatalf("registerBotCommands() error: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != "/me/commands" {
		t.Errorf("path = %q, want /me/commands", gotPath)
	}
	if gotAuth != "bot-token" {
		t.Errorf("Authorization = %q, want bot-token", gotAuth)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotCT)
	}

	var payload struct {
		Commands []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("invalid JSON body %q: %v", gotBody, err)
	}

	// Все команды, которые обрабатывает internal/handler/bot.go, должны
	// попадать в подсказки MAX.
	want := map[string]bool{
		"start": false, "token": false, "newtoken": false, "status": false,
		"heartbeat": false, "bind": false, "groupid": false, "help": false,
	}
	for _, c := range payload.Commands {
		if _, ok := want[c.Name]; !ok {
			t.Errorf("unexpected command %q", c.Name)
			continue
		}
		if c.Description == "" {
			t.Errorf("command %q has empty description", c.Name)
		}
		want[c.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("command %q is not registered", name)
		}
	}
}

func TestRegisterBotCommandsReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"token revoked"}`))
	}))
	defer srv.Close()

	err := registerBotCommands(srv.URL, "bad-token")
	if err == nil {
		t.Fatal("registerBotCommands() = nil error, want error on HTTP 401")
	}
	if got := err.Error(); !strings.Contains(got, "401") {
		t.Errorf("error = %q, want it to mention HTTP 401", got)
	}
}
