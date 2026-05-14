package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"gitflic.ru/piroman99/2mon/internal/config"
	"gitflic.ru/piroman99/2mon/internal/handler"
	"gitflic.ru/piroman99/2mon/internal/notifier"
	"gitflic.ru/piroman99/2mon/internal/scheduler"
	"gitflic.ru/piroman99/2mon/internal/sender"
	"gitflic.ru/piroman99/2mon/internal/store"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Загружаем конфиг
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("конфиг: %v", err)
	}

	// Подключаемся к MongoDB
	ctx := context.Background()
	st, err := store.New(ctx, cfg.MongoURI)
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	defer st.Close(ctx)
	log.Println("mongo: подключено")

	// Создаём sender
	snd := sender.New(cfg.MaxBotToken, cfg.MaxAPIURL, cfg.RateLimit, cfg.QueueSize)

	// Запускаем sender
	senderCtx, senderCancel := context.WithCancel(ctx)
	defer senderCancel()
	snd.Start(senderCtx)
	log.Println("sender: запущен")

	// Создаём notifier
	ntf := notifier.New(st, snd)

	// Создаём scheduler
	sch := scheduler.New(st, snd, ntf)

	// Запускаем scheduler
	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()
	go sch.Start(schedCtx)

	// Создаём обработчики
	webhookH := handler.NewWebhookHandler(st, snd)
	botH := handler.NewBotHandler(st, snd, ntf)
	adminH := handler.NewAdminHandler(st, cfg.AdminPassword)

	// Настраиваем роутер
	r := chi.NewRouter()

	// Вебхуки от Zabbix
	r.Post("/wh/{token}", webhookH.Handle)

	// Вебхуки от MAX (бот)
	r.Post("/bot", botH.Handle)

	// Админка
	r.Get("/admin", adminH.LoginPage)
	r.Post("/admin/login", adminH.Login)
	r.Get("/admin/dashboard", adminH.Dashboard)
	r.Post("/admin/ban/{chatID}", adminH.BanUser)
	r.Post("/admin/limit/{chatID}", adminH.SetLimit)

	// Запускаем HTTP-сервер
	addr := ":" + cfg.Port
	log.Printf("сервер: запущен на %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, r); err != nil {
			log.Fatalf("сервер: %v", err)
		}
	}()


        // Установка команд бота при старте
        go func() {
                body := `{"commands":[{"name":"start","description":"Регистрация"},{"name":"token","description":"Показать токен"},{"name":"status","description":"Статистика за сегодня"},{"name":"heartbeat","description":"Статус heartbeat"},{"name":"help","description":"Справка"}]}`
                req, _ := http.NewRequest("PATCH", cfg.MaxAPIURL+"/me", strings.NewReader(body))
                req.Header.Set("Authorization", cfg.MaxBotToken)
                req.Header.Set("Content-Type", "application/json")
                resp, err := http.DefaultClient.Do(req)
                if err != nil {
                        log.Printf("[commands] ошибка: %v", err)
                        return
                }
                resp.Body.Close()
                log.Println("[commands] команды бота обновлены")
        }()

	// Ждём сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("завершение...")
	senderCancel()
	schedCancel()
	log.Println("пока!")
}
