package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"gitflic.ru/piroman99/2mon/internal/model"
	"gitflic.ru/piroman99/2mon/internal/notifier"
	"gitflic.ru/piroman99/2mon/internal/sender"
	"gitflic.ru/piroman99/2mon/internal/store"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	store    *store.Store
	sender   *sender.Sender
	notifier *notifier.Notifier
	cron     *cron.Cron
}

func New(s *store.Store, snd *sender.Sender, n *notifier.Notifier) *Scheduler {
	return &Scheduler{
		store:    s,
		sender:   snd,
		notifier: n,
		cron:     cron.New(),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.cron.AddFunc("0 9 * * 1-5", s.sendStatsToUsers)
	s.cron.AddFunc("5 9 * * 1-5", s.sendStatsToAdmins)
	s.cron.Start()
	log.Println("[scheduler] запущен")
	<-ctx.Done()
	s.cron.Stop()
	log.Println("[scheduler] остановлен")
}

func (s *Scheduler) sendStatsToUsers() {
	users, err := s.store.FindActiveUsers(context.Background())
	if err != nil {
		log.Printf("[scheduler] find users: %v", err)
		return
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	adText := "\n\n📢 Реклама\nВаш надёжный прокси для уведомлений — 2mon."
	for _, user := range users {
		count, _ := s.store.CountMessagesByDate(context.Background(), yesterday)
		text := fmt.Sprintf("📊 Ваша статистика за %s\n\nОтправлено сообщений: %d\nДневной лимит: %d%s", yesterday, count, user.DailyLimit, adText)
		s.sender.Enqueue(model.Message{ChatID: user.ChatID, Text: text})
	}
}

func (s *Scheduler) sendStatsToAdmins() {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	total, _ := s.store.CountMessagesByDate(context.Background(), yesterday)
	newUsers, _ := s.store.CountUsersByDate(context.Background(), yesterday)
	top, _ := s.store.TopUsersByMessages(context.Background(), yesterday, 5)
	text := fmt.Sprintf("📊 Статистика за %s\n\nСообщений: %d\nНовых пользователей: %d\n", yesterday, total, newUsers)
	if len(top) > 0 {
		text += "\n🏆 Топ пользователей:\n"
		medals := []string{"🥇", "🥈", "🥉", "▫️", "▫️"}
		for i, u := range top {
			text += fmt.Sprintf("%s %d сообщ.\n", medals[i], u.MsgCount)
		}
	}
	s.notifier.NotifyAdmins(text)
}
