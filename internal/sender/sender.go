package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"gitflic.ru/piroman99/2mon/internal/model"

	"golang.org/x/time/rate"
)

type Sender struct {
	botToken string
	apiURL   string
	limiter  *rate.Limiter
	queue    chan model.Message
	closed   atomic.Bool
}

func New(botToken, apiURL string, rateLimit, queueSize int) *Sender {
	return &Sender{
		botToken: botToken,
		apiURL:   apiURL,
		limiter:  rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
		queue:    make(chan model.Message, queueSize),
	}
}

// Start запускает обработчик очереди. При отмене ctx очередь
// дорабатывается (drain), после чего новые сообщения не принимаются.
func (s *Sender) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case msg := <-s.queue:
				s.dispatch(ctx, msg)
			case <-ctx.Done():
				log.Println("[sender] draining queue...")
				// Запрещаем Enqueue дорабатывать очередь после старта
				// завершения, но канал НЕ закрываем: его могут писать
				// конкурентные HTTP-обработчики.
				s.closed.Store(true)
				s.drainQueue()
				return
			}
		}
	}()
}

func (s *Sender) dispatch(ctx context.Context, msg model.Message) {
	if err := s.limiter.Wait(ctx); err != nil {
		// Отменённый контекст не должен ронять доставку — логируем и шлём.
		log.Printf("[sender] limiter wait: %v", err)
	}
	log.Printf("[sender] to %s: %s", msg.ChatID, truncate(msg.Text, 50))
	if err := s.sendToMax(ctx, msg); err != nil {
		log.Printf("[sender] send error to %s: %v", msg.ChatID, err)
	}
}

func (s *Sender) drainQueue() {
	for {
		select {
		case msg := <-s.queue:
			s.dispatch(context.Background(), msg)
		default:
			return
		}
	}
}

func (s *Sender) Enqueue(msg model.Message) error {
	if s.closed.Load() {
		return fmt.Errorf("sender остановлен")
	}
	select {
	case s.queue <- msg:
		return nil
	default:
		return fmt.Errorf("очередь переполнена")
	}
}

func (s *Sender) QueueLen() int {
	return len(s.queue)
}

func (s *Sender) sendToMax(ctx context.Context, msg model.Message) error {
	url := fmt.Sprintf("%s/messages?chat_id=%s", s.apiURL, msg.ChatID)

	body := map[string]string{
		"text": msg.Text,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", s.botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("max api returned %d", resp.StatusCode)
	}

	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
