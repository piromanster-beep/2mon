package handler

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
	"gitflic.ru/piroman99/2mon/internal/store"
)

// AdminHandler — обработчик админки
type AdminHandler struct {
	store    *store.Store
	password string
	tmpl     *template.Template
}

// NewAdminHandler — создать обработчик админки
func NewAdminHandler(s *store.Store, password string) *AdminHandler {
	tmpl := template.Must(template.ParseFiles("web/admin.html"))
	return &AdminHandler{
		store:    s,
		password: password,
		tmpl:     tmpl,
	}
}

// LoginPage — страница входа
func (h *AdminHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginHTML := `<!DOCTYPE html>
<html>
<head><title>2mon — вход</title></head>
<body>
<h1>2mon Admin</h1>
<form method="POST" action="/admin/login">
<input type="password" name="password" placeholder="Пароль">
<button type="submit">Войти</button>
</form>
</body>
</html>`
	w.Write([]byte(loginHTML))
}

// Login — проверка пароля
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")

	if password != h.password {
		w.Write([]byte("Неверный пароль. <a href='/admin'>Назад</a>"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "admin_session",
		Value: h.password,
		Path:  "/",
	})

	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

// Dashboard — основная страница админки
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	if !h.checkAuth(r) {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	users, err := h.store.FindActiveUsers(r.Context())
	if err != nil {
		http.Error(w, "ошибка загрузки пользователей", http.StatusInternalServerError)
		return
	}

	// Отдельно ищем админов (могут быть неактивны)
	admins, _ := h.store.FindAdmins(r.Context())
	blocked, _ := h.store.FindBlockedUsers(r.Context())
	selfBlocked := h.getSelfBlockedUsers()


	// Вычисляем статус heartbeat
	now := time.Now()
	for i := range users {
		if !users[i].LastHeartbeat.IsZero() {
			timeout := time.Duration(users[i].HeartbeatInterval+10) * time.Minute
			if users[i].HeartbeatInterval == 0 {
				timeout = 15 * time.Minute
			}
			users[i].HeartbeatOK = now.Sub(users[i].LastHeartbeat) < timeout
		}
	}

		data := struct {
		Users   interface{}
		Admins  interface{}
		Blocked interface{}
		SelfBlocked interface{}
	}{
		Users:   users,
		Admins:  admins,
		Blocked: blocked,
		SelfBlocked: selfBlocked,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.Execute(w, data)
}

// BanUser — заблокировать/разблокировать пользователя
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	if !h.checkAuth(r) {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	// Из URL: /admin/ban/chat_id
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "не указан chat_id", http.StatusBadRequest)
		return
	}
	chatID := parts[3]

	user, err := h.store.FindByChatID(r.Context(), chatID)
	if err != nil || user == nil {
		http.Error(w, "пользователь не найден", http.StatusNotFound)
		return
	}

	user.IsActive = !user.IsActive
	h.store.UpdateUser(r.Context(), user)

	log.Printf("[admin] пользователь %s: is_active = %v", chatID, user.IsActive)
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

// SetLimit — изменить лимит пользователю
func (h *AdminHandler) SetLimit(w http.ResponseWriter, r *http.Request) {
	if !h.checkAuth(r) {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	// Из URL: /admin/limit/chat_id
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "не указан chat_id", http.StatusBadRequest)
		return
	}
	chatID := parts[3]

	newLimit := r.FormValue("limit")
	if newLimit == "" {
		http.Error(w, "не указан лимит", http.StatusBadRequest)
		return
	}

	user, err := h.store.FindByChatID(r.Context(), chatID)
	if err != nil || user == nil {
		http.Error(w, "пользователь не найден", http.StatusNotFound)
		return
	}

	// Парсим лимит
	var limit int
	if _, err := fmt.Sscanf(newLimit, "%d", &limit); err != nil {
		http.Error(w, "неверный формат лимита", http.StatusBadRequest)
		return
	}

	user.DailyLimit = limit
	h.store.UpdateUser(r.Context(), user)

	log.Printf("[admin] пользователь %s: новый лимит = %d", chatID, limit)
	http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
}

// checkAuth — проверить куку сессии
func (h *AdminHandler) checkAuth(r *http.Request) bool {
	cookie, err := r.Cookie("admin_session")
	if err != nil {
		return false
	}
	return cookie.Value == h.password
}

//разбор логов в поисках самоблока
func (h *AdminHandler) getSelfBlockedUsers() []string {
	cmd := exec.Command("docker", "logs", "2mon", "--tail", "500")
	output, _ := cmd.Output()
	lines := strings.Split(string(output), "\n")

	blockedMap := make(map[string]bool)
	for _, line := range lines {
		if strings.Contains(line, "max api returned 403") {
			// Извлекаем chat_id из строки вида "chat_id=-74740811238662"
			if idx := strings.Index(line, "chat_id="); idx != -1 {
				chatID := line[idx+8:]
				if end := strings.Index(chatID, "\""); end != -1 {
					chatID = chatID[:end]
				}
				if end := strings.Index(chatID, " "); end != -1 {
					chatID = chatID[:end]
				}
				blockedMap[chatID] = true
			}
		}
	}

	var result []string
	for chatID := range blockedMap {
		result = append(result, chatID)
	}
	return result
}
