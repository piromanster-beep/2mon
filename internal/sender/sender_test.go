package sender

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gitflic.ru/piroman99/2mon/internal/model"
)

func okServer(t *testing.T, hits *int32, mu *sync.Mutex) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token" {
			t.Errorf("Authorization = %q, want token", got)
		}
		mu.Lock()
		*hits++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func waitClosed(t *testing.T, s *Sender) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if s.closed.Load() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("sender did not reach closed state in time")
}

func TestDeliversQueuedMessage(t *testing.T) {
	var mu sync.Mutex
	var hits int32
	srv := okServer(t, &hits, &mu)

	s := New("token", srv.URL, 100, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	if err := s.Enqueue(model.Message{ChatID: "1", Text: "hi"}); err != nil {
		t.Fatalf("Enqueue() error: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := hits
		mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("queued message was not delivered")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestEnqueueAfterShutdownReturnsError(t *testing.T) {
	var mu sync.Mutex
	var hits int32
	srv := okServer(t, &hits, &mu)

	s := New("token", srv.URL, 100, 10)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	cancel()
	waitClosed(t, s)

	if err := s.Enqueue(model.Message{ChatID: "1", Text: "late"}); err == nil {
		t.Fatal("Enqueue() after shutdown = nil error, want error")
	}
}

// Регрессия: Enqueue() вызывается из конкурентных HTTP-обработчиков,
// а shutdown дренирует очередь. Раньше drainQueue() делал close(queue),
// что давало panic "send on closed channel".
func TestConcurrentEnqueueDuringShutdown(t *testing.T) {
	var mu sync.Mutex
	var hits int32
	srv := okServer(t, &hits, &mu)

	s := New("token", srv.URL, 1000, 256)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	var wg sync.WaitGroup
	stop := time.Now().Add(400 * time.Millisecond)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(stop) {
				_ = s.Enqueue(model.Message{ChatID: "1", Text: "x"})
			}
		}()
	}

	// Отменяем контекст ПОКА enqueue-горутины ещё пишут в очередь:
	// именно в это окно старый close(queue) в drainQueue() давал
	// panic "send on closed channel".
	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()
	waitClosed(t, s)
}
