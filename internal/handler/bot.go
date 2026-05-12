package handler

import (
	"encoding/json"
	"fmt"
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
	var update struct {
		Message struct {
			Chat struct {
				ID string `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
		} `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)
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
	return fmt.Sprintf("Привет! Вы зарегистрированы.\n\nВаш токен: %s\n\nИспользуйте его для настройки вебхука в Zabbix:\nhttps://ваш-сервер/wh/%s\n\nКоманды:\n/status — статистика\n/token — показать токен\n/help — справка", token, token)
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
