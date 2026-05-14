package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"gitflic.ru/piroman99/2mon/internal/model"
	"gitflic.ru/piroman99/2mon/internal/sender"
	"gitflic.ru/piroman99/2mon/internal/store"
)

// WebhookHandler — обработчик вебхуков от внешних сервисов
type WebhookHandler struct {
	store  *store.Store
	sender *sender.Sender
}

// NewWebhookHandler — создать обработчик
func NewWebhookHandler(s *store.Store, sender *sender.Sender) *WebhookHandler {
	return &WebhookHandler{store: s, sender: sender}
}

// Handle — обработать POST /wh/{token}
func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Достаём токен из URL
	// URL выглядит как /wh/sec_abc123
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 || parts[1] != "wh" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	token := parts[2]

	// Ищем пользователя
	user, err := h.store.FindByToken(r.Context(), token)
	if err != nil {
		log.Printf("[webhook] find user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "invalid token", http.StatusNotFound)
		return
	}

	// Проверяем, активен ли
	if !user.IsActive {
		http.Error(w, "user disabled", http.StatusForbidden)
		return
	}

	// Парсим тело
	var payload model.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Проверяем лимит
	count, err := h.store.IncrementMsgCount(r.Context(), user)
	if err != nil {
		log.Printf("[webhook] increment count: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if count > user.DailyLimit {
		// Лимит превышен — логируем и отказываем
		h.store.LogMessage(r.Context(), &model.MessageLog{
			UserID: user.ID,
			Source: "zabbix",
			Status: "limit_exceeded",
		})

		// Уведомляем пользователя один раз в день
		if count == user.DailyLimit+1 {
			h.sender.Enqueue(model.Message{
				ChatID: user.ChatID,
				Text:   fmt.Sprintf("⚠️ Дневной лимит сообщений исчерпан (%d/%d). Лимит сбросится в полночь.", user.DailyLimit, user.DailyLimit),
			})
		}

		http.Error(w, "daily limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Формируем текст для MAX
	text := formatMessage(payload)

	// Кладём в очередь на отправку
	if err := h.sender.Enqueue(model.Message{
		ChatID: user.ChatID,
		Text:   text,
	}); err != nil {
		log.Printf("[webhook] enqueue: %v", err)
		http.Error(w, "queue full", http.StatusTooManyRequests)
		return
	}

	// Логируем успех
	h.store.LogMessage(r.Context(), &model.MessageLog{
		UserID: user.ID,
		Source: "zabbix",
		Status: "success",
	})

	// Ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

// formatMessage — красивое оформление сообщения
func formatMessage(p model.WebhookPayload) string {
	// Эмодзи по severity
	emoji := "ℹ️"
	switch strings.ToLower(p.Severity) {
	case "warning", "average":
		emoji = "⚠️"
	case "high", "critical", "disaster":
		emoji = "🔴"

	case "information", "info":
		emoji = "ℹ️"
	}

	return fmt.Sprintf("%s *%s*\n%s", emoji, p.Subject, p.Message)
}
