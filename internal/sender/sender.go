package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"gitflic.ru/piroman99/2mon/internal/model"

	"golang.org/x/time/rate"
)

const (
	// sendTimeout — потолок на один HTTP-запрос к MAX.
	sendTimeout = 10 * time.Second
	// maxAttempts — сколько раз пробуем доставить при временных ошибках
	// (429 и 5xx). Не-временные ответы (403 и прочие 4xx) не повторяем,
	// чтобы не спамить пользователя дублями.
	maxAttempts = 3
	// defaultRetryDelay — пауза перед повтором, если MAX не прислал
	// Retry-After; растёт с номером попытки.
	defaultRetryDelay = time.Second
)

// ErrorReporter сообщает результат попытки отправки в MAX.
// err == nil — доставка успешна; иначе err описывает причину
// (например HTTP 403, если бот заблокирован пользователем).
type ErrorReporter func(chatID string, err error)

type Sender struct {
	botToken string
	apiURL   string
	limiter  *rate.Limiter
	queue    chan model.Message
	closed   atomic.Bool
	reporter ErrorReporter
	client   *http.Client
}

func New(botToken, apiURL string, rateLimit, queueSize int) *Sender {
	return &Sender{
		botToken: botToken,
		apiURL:   apiURL,
		limiter:  rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
		queue:    make(chan model.Message, queueSize),
		// Один клиент на весь sender: соединения переиспользуются
		// (keep-alive), а не пересоздаются на каждое сообщение.
		client: &http.Client{
			Timeout: sendTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// SetErrorReporter включает фиксацию результата каждой отправки.
// Вызывать до Start (поле читается только из горутины sender).
func (s *Sender) SetErrorReporter(r ErrorReporter) {
	s.reporter = r
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
	err := s.sendToMax(ctx, msg)
	if err != nil {
		log.Printf("[sender] send error to %s: %v", msg.ChatID, err)
	}
	if s.reporter != nil {
		s.reporter(msg.ChatID, err)
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

// sendToMax отправляет сообщение, повторяя попытку при временных ошибках
// MAX (429 Too Many Requests и 5xx). Остальные ответы — окончательный
// результат: их возвращаем сразу, чтобы не задваивать уведомления.
func (s *Sender) sendToMax(ctx context.Context, msg model.Message) error {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		retry, err := s.postMessage(ctx, msg)
		if err == nil {
			return nil
		}
		lastErr = err

		// Повторяем только временные ошибки и только если контекст жив.
		if retry <= 0 || attempt == maxAttempts || ctx.Err() != nil {
			break
		}
		log.Printf("[sender] retry %d/%d to %s in %s: %v", attempt, maxAttempts, msg.ChatID, retry, err)
		select {
		case <-ctx.Done():
			return lastErr
		case <-time.After(retry):
		}
	}
	return lastErr
}

// postMessage — одна попытка отправки. Если ответ временный (429/5xx),
// возвращает паузу перед повтором, иначе 0.
func (s *Sender) postMessage(ctx context.Context, msg model.Message) (time.Duration, error) {
	url := fmt.Sprintf("%s/messages?chat_id=%s", s.apiURL, msg.ChatID)

	body := map[string]string{
		"text": msg.Text,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", s.botToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		// Неизвестно, дошло ли сообщение: повторять рискованно
		// (возможен дубль), поэтому считаем ошибку окончательной.
		return 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	// Вычитываем тело: без этого соединение не вернётся в пул keep-alive.
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	_, _ = io.Copy(io.Discard, resp.Body)

	switch {
	case resp.StatusCode == http.StatusOK:
		return 0, nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return retryDelay(resp, 1), fmt.Errorf("max api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	case resp.StatusCode >= 500:
		return retryDelay(resp, 2), fmt.Errorf("max api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	default:
		return 0, fmt.Errorf("max api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}
}

// retryDelay берёт паузу из Retry-After (в секундах), иначе —
// defaultRetryDelay, умноженный на номер попытки.
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if v := strings.TrimSpace(resp.Header.Get("Retry-After")); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(attempt) * defaultRetryDelay
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
