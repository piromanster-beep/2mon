package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"gitflic.ru/piroman99/2mon/internal/model"
	"gitflic.ru/piroman99/2mon/internal/notifier"
	"gitflic.ru/piroman99/2mon/internal/sender"
	"gitflic.ru/piroman99/2mon/internal/store"

	"github.com/google/uuid"
)

type BotHandler struct {
	store    *store.Store
	sender   *sender.Sender
	notifier *notifier.Notifier
}

func NewBotHandler(s *store.Store, snd *sender.Sender, n *notifier.Notifier) *BotHandler {
	return &BotHandler{store: s, sender: snd, notifier: n}
}

func (h *BotHandler) Handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	log.Printf("[bot] raw update: %s", string(body))

	// Структура MAX: message.recipient.chat_id, message.body.text
	var update struct {
		Message struct {
			Recipient struct {
				ChatID   int64  `json:"chat_id"`
				ChatType string `json:"chat_type"`
				UserID   int64  `json:"user_id"`
			} `json:"recipient"`
			Body struct {
				Text string `json:"text"`
			} `json:"body"`
			Sender struct {
				UserID    int64  `json:"user_id"`
				FirstName string `json:"first_name"`
			} `json:"sender"`
		} `json:"message"`
	}

	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("[bot] parse error: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	chatID := fmt.Sprintf("%d", update.Message.Recipient.ChatID)
	text := strings.TrimSpace(update.Message.Body.Text)

	log.Printf("[bot] chat_id=%s, text=%s", chatID, text)

	var response string
	switch {
	case text == "/start":
		response = h.handleStart(r, chatID)
	case text == "/token":
		response = h.handleToken(r, chatID)
	case text == "/status":
		response = h.handleStatus(r, chatID)
	default:
		response = "Неизвестная команда. Напишите /help"
	}

	log.Printf("[bot] response to %s: %s", chatID, response)
	h.sender.Enqueue(model.Message{ChatID: chatID, Text: response})
	w.WriteHeader(http.StatusOK)
}

func (h *BotHandler) handleStart(r *http.Request, chatID string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user != nil {
		return fmt.Sprintf("Вы уже зарегистрированы!\nВаш токен: %s\n\nСтатистика: /status", user.Token)
	}
	token := uuid.New().String()
	user = &model.User{
		ChatID:     chatID,
		Token:      token,
		IsActive:   true,
		IsAdmin:    false,
		DailyLimit: 100,
	}
	h.store.CreateUser(r.Context(), user)
	h.notifier.NotifyAdmins(fmt.Sprintf("🆕 Новый пользователь\nChat ID: %s", chatID))
	return fmt.Sprintf("Привет! Вы зарегистрированы.\n\nВаш токен: %s\n\nИспользуйте его для настройки вебхука в Zabbix:\nhttps://2mon.ru/wh/%s\n\nКоманды:\n/status — статистика\n/token — показать токен\n/help — справка", token, token)
}

func (h *BotHandler) handleToken(r *http.Request, chatID string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user == nil {
		return "Вы не зарегистрированы. Напишите /start"
	}
	return fmt.Sprintf("Ваш токен: %s", user.Token)
}

func (h *BotHandler) handleStatus(r *http.Request, chatID string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user == nil {
		return "Вы не зарегистрированы. Напишите /start"
	}
	return fmt.Sprintf("📊 Статистика за сегодня\n\nОтправлено: %d / %d\nОсталось: %d", user.MsgCountToday, user.DailyLimit, user.DailyLimit-user.MsgCountToday)
}
