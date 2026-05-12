package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"gitflic.ru/piroman99/2mon/internal/model"

	"golang.org/x/time/rate"
)

// Sender — отправщик сообщений в MAX
type Sender struct {
	botToken string
	apiURL   string
	limiter  *rate.Limiter
	queue    chan model.Message
}

// New — создать новый Sender
func New(botToken, apiURL string, rateLimit, queueSize int) *Sender {
	return &Sender{
		botToken: botToken,
		apiURL:   apiURL,
		limiter:  rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
		queue:    make(chan model.Message, queueSize),
	}
}

// Start — запустить воркер обработки очереди
func (s *Sender) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case msg := <-s.queue:
				// Ждём разрешения rate limiter'а
				if err := s.limiter.Wait(ctx); err != nil {
					log.Printf("[sender] limiter wait: %v", err)
					continue
				}
				// Отправляем в MAX
				if err := s.sendToMax(ctx, msg); err != nil {
					log.Printf("[sender] send error: %v", err)
				}
			case <-ctx.Done():
				log.Println("[sender] draining queue...")
				s.drainQueue()
				return
			}
		}
	}()
}

// drainQueue — опустошить очередь перед выключением
func (s *Sender) drainQueue() {
	close(s.queue)
	for msg := range s.queue {
		if err := s.sendToMax(context.Background(), msg); err != nil {
			log.Printf("[sender] drain error: %v", err)
		}
	}
}

// Enqueue — положить сообщение в очередь
func (s *Sender) Enqueue(msg model.Message) error {
	select {
	case s.queue <- msg:
		return nil
	default:
		return fmt.Errorf("очередь переполнена")
	}
}

// QueueLen — текущий размер очереди
func (s *Sender) QueueLen() int {
	return len(s.queue)
}

// sendToMax — HTTP-запрос к MAX API
func (s *Sender) sendToMax(ctx context.Context, msg model.Message) error {
	url := fmt.Sprintf("%s/bot%s/sendMessage", s.apiURL, s.botToken)

	body := map[string]string{
		"chat_id": msg.ChatID,
		"text":    msg.Text,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
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
