package handler

import (
	"encoding/json"
	"fmt"
	"gitflic.ru/piroman99/2mon/internal/model"
	"gitflic.ru/piroman99/2mon/internal/notifier"
	"gitflic.ru/piroman99/2mon/internal/sender"
	"gitflic.ru/piroman99/2mon/internal/store"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

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
	//debug log.Printf("[bot] raw update: %s", string(body))

	// Структура MAX: message.recipient.chat_id, message.body.text
	var update struct {
		UpdateType string `json:"update_type"`
		Message    struct {
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

	var chatID string
	var text string
	//debug log.Printf("[bot] update_type=%s", update.UpdateType)

	if update.UpdateType == "bot_added" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if update.Message.Recipient.ChatID == 0 {
		var raw struct {
			ChatID int64 `json:"chat_id"`
		}
		json.Unmarshal(body, &raw)
		if raw.ChatID != 0 {
			chatID = fmt.Sprintf("%d", raw.ChatID)
			text = "/start"
		}
	} else {
		chatID = fmt.Sprintf("%d", update.Message.Recipient.ChatID)
		text = strings.TrimSpace(update.Message.Body.Text)
	}

	//debug	log.Printf("[bot] chat_id=%s, text=%s", chatID, text)

	var response string
	// Если сообщение начинается с @ — ищем команду после первого пробела
	if strings.HasPrefix(text, "@") {
		if idx := strings.Index(text, " "); idx != -1 {
			text = text[idx+1:]
		}
	}
	switch {
	case text == "/start":
		if strings.HasPrefix(chatID, "-") {
			response = fmt.Sprintf("Группа зарегистрирована. Для получения токена напишите боту в личку: /bind %s", chatID)
		} else {
			response = h.handleStart(r, chatID)
		}
	case text == "/token":
		// В группах токен не показываем
		if strings.HasPrefix(chatID, "-") {
			response = "Токен можно получить только в личных сообщениях. Напишите боту в личку."
		} else {
			response = h.handleToken(r, chatID)
		}
	case text == "/status":
		response = h.handleStatus(r, chatID)
	case text == "/help":
		response = "Доступные команды:\n\n/status — статистика за сегодня\n/heartbeat — статус heartbeat\n/token — показать токен\n/newtoken — сменить токен\n/bind — привязать группу\n/groupid — ID группы\n/help — справка"
	case text == "/heartbeat":
		response = h.handleHeartbeat(r, chatID)
	case strings.HasPrefix(text, "/newtoken"):
		// Смена токена — только в личке
		if strings.HasPrefix(chatID, "-") {
			response = "Сменить токен можно только в личных сообщениях. Напишите боту в личку."
		} else {
			response = h.handleNewToken(r, chatID, text)
		}
	case strings.HasPrefix(text, "/bind"):
		if strings.HasPrefix(chatID, "-") {
			w.WriteHeader(http.StatusOK)
			return
		}
		response = h.handleBind(r, chatID, text)
	case text == "/groupid":
		response = fmt.Sprintf("ID этой группы: %s", chatID)
	default:
		if strings.HasPrefix(chatID, "-") {
			if !strings.HasPrefix(text, "/") {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
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
	return fmt.Sprintf("Привет! Вы зарегистрированы.\n\nВаш токен: %s\n\nИспользуйте его для настройки вебхука в Zabbix:\nhttps://2mon.ru/wh/%s\n\nКоманды:\n/status — статистика\n/token — показать токен\n/newtoken — сменить токен\n/help — справка", token, token)
}

func (h *BotHandler) handleToken(r *http.Request, chatID string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user == nil {
		return "Вы не зарегистрированы. Напишите /start"
	}
	return fmt.Sprintf("Ваш токен: %s", user.Token)
}

// handleNewToken меняет (ротирует) токен пользователя.
// Без слова confirm — только предупреждение, токен не трогаем.
func (h *BotHandler) handleNewToken(r *http.Request, chatID, text string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user == nil {
		return "Вы не зарегистрированы. Напишите /start"
	}

	parts := strings.Fields(text)
	if len(parts) < 2 || parts[1] != "confirm" {
		return "⚠️ Смена токена отключит текущий вебхук.\n" +
			"Zabbix перестанет слать уведомления, пока вы не подставите новый токен в настройках медиа-типа.\n\n" +
			"Чтобы продолжить, отправьте:\n/newtoken confirm"
	}

	user.Token = uuid.New().String()
	if err := h.store.UpdateUser(r.Context(), user); err != nil {
		log.Printf("[bot] newtoken update %s: %v", chatID, err)
		return "Не удалось сменить токен, попробуйте позже."
	}

	h.notifier.NotifyAdmins(fmt.Sprintf("🔑 Пользователь %s сменил токен", chatID))

	return fmt.Sprintf("✅ Токен обновлён. Старый токен больше не работает.\n\n"+
		"Новый токен: %s\n\n"+
		"URL для Zabbix:\nhttps://2mon.ru/wh/%s\n\n"+
		"Не забудьте обновить токен в медиа-типе Zabbix, иначе уведомления не будут приходить.",
		user.Token, user.Token)
}

func (h *BotHandler) handleStatus(r *http.Request, chatID string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user == nil {
		return "Вы не зарегистрированы. Напишите /start"
	}
	return fmt.Sprintf("📊 Статистика за сегодня\n\nОтправлено: %d / %d\nОсталось: %d", user.MsgCountToday, user.DailyLimit, user.DailyLimit-user.MsgCountToday)
}

func (h *BotHandler) handleHeartbeat(r *http.Request, chatID string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user == nil {
		return "Вы не зарегистрированы. Напишите /start"
	}

	if user.LastHeartbeat.IsZero() {
		return "❤️ Heartbeat ещё не настроен.\n\nДобавьте в Zabbix Action, который шлёт вебхук с subject=heartbeat на ваш URL.\nИнтервал: раз в 5 минут."
	}

	//	ago := time.Since(user.LastHeartbeat).Round(time.Minute)
	//	return fmt.Sprintf("❤️ Heartbeat: OK\nПоследний сигнал: %s назад", ago)

	ago := time.Since(user.LastHeartbeat).Round(time.Minute)
	timeout := time.Duration(user.HeartbeatInterval+10) * time.Minute
	if user.HeartbeatInterval == 0 {
		timeout = 15 * time.Minute
	}

	if ago > timeout {
		return fmt.Sprintf("🔴 Heartbeat: нет сигнала\nПоследний сигнал: %s назад\nТаймаут: %s", ago, timeout)
	}

	return fmt.Sprintf("🟢 Heartbeat: OK\nПоследний сигнал: %s назад", ago)
}
func (h *BotHandler) handleBind(r *http.Request, chatID string, text string) string {
	user, _ := h.store.FindByChatID(r.Context(), chatID)
	if user == nil {
		return "Вы не зарегистрированы. Напишите /start в личных сообщениях."
	}

	// /bind GROUP_ID или /bind
	parts := strings.Fields(text)
	if len(parts) < 2 {
		// Показываем текущую привязку
		if user.GroupChatID != "" {
			return fmt.Sprintf("Бот привязан к группе %s.\nЧтобы отвязать: /bind off", user.GroupChatID)
		}
		return "Укажите ID группы: /bind GROUP_ID\nЧтобы отвязать: /bind off\n\nID группы можно узнать, добавив бота в группу и написав /groupid (в группе)."
	}

	arg := parts[1]

	// Отвязка
	if arg == "off" {
		if user.GroupChatID == "" {
			return "Бот и так не привязан к группе."
		}
		user.GroupChatID = ""
		if err := h.store.UpdateUser(r.Context(), user); err != nil {
			log.Printf("[bot] unbind %s: %v", chatID, err)
			return "Не удалось отвязать группу, попробуйте позже."
		}
		return "Бот отвязан от группы. Уведомления снова пойдут в личные сообщения."
	}

	// Привязка
	user.GroupChatID = arg
	if err := h.store.UpdateUser(r.Context(), user); err != nil {
		log.Printf("[bot] bind %s -> %s: %v", chatID, arg, err)
		return "Не удалось привязать группу, попробуйте позже."
	}

	// Если группа уже зарегистрирована (/start в группе), показываем её токен,
	// иначе — личный токен пользователя.
	groupToken := user.Token
	if groupUser, _ := h.store.FindByChatID(r.Context(), arg); groupUser != nil {
		groupToken = groupUser.Token
	}

	return fmt.Sprintf("Бот привязан к группе %s. Уведомления будут приходить туда.\n\nТокен для настройки Zabbix: %s", arg, groupToken)
}
