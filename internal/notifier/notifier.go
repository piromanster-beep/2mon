package notifier

import (
	"context"
	"log"

	"gitflic.ru/piroman99/2mon/internal/model"
	"gitflic.ru/piroman99/2mon/internal/sender"
	"gitflic.ru/piroman99/2mon/internal/store"
)

// Notifier — уведомления администраторам
type Notifier struct {
	store  *store.Store
	sender *sender.Sender
}

// New — создать Notifier
func New(s *store.Store, sender *sender.Sender) *Notifier {
	return &Notifier{store: s, sender: sender}
}

// NotifyAdmins — отправить сообщение всем админам
func (n *Notifier) NotifyAdmins(text string) {
	admins, err := n.store.FindAdmins(context.Background())
	if err != nil {
		log.Printf("[notifier] find admins: %v", err)
		return
	}

	for _, admin := range admins {
		if err := n.sender.Enqueue(model.Message{
			ChatID: admin.ChatID,
			Text:   text,
		}); err != nil {
			log.Printf("[notifier] enqueue to admin %s: %v", admin.ChatID, err)
		}
	}
}