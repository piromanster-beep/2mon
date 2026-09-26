package store

import (
	"context"
	"os"
	"testing"
	"time"

	"gitflic.ru/piroman99/2mon/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// getTestURI возвращает URI для тестов. Использует MONGO_TEST_URI,
// иначе стандартный localhost.
func getTestURI() string {
	if uri := os.Getenv("MONGO_TEST_URI"); uri != "" {
		return uri
	}
	return "mongodb://localhost:27017"
}

// setupStore создаёт Store с изолированной тестовой коллекцией users.
func setupStore(t *testing.T) (*Store, *mongo.Collection) {
	t.Helper()
	uri := getTestURI()
	s, err := New(context.Background(), uri, 7)
	if err != nil {
		t.Skipf("пропуск: mongo недоступна (%v)", err)
	}

	// Уникальная коллекция на каждый тест — изоляция между запусками.
	// Используем публичное поле db напрямую.
	colName := "test_2mon_" + t.Name()
	col := s.db.Collection(colName)

	t.Cleanup(func() {
		ctx := context.Background()
		_ = col.Drop(ctx)
		_ = s.Close(ctx)
	})

	return s, col
}

// TestMarkUserSendError_SetsFields — обычная ошибка не банит.
func TestMarkUserSendError_SetsFields(t *testing.T) {
	ctx := context.Background()
	s, col := setupStore(t)
	chatID := "111222333"

	// Создаём пользователя через store, чтобы получить валидный model.User
	user := &model.User{
		ChatID:   chatID,
		Token:    "tok_" + chatID,
		IsActive: true,
	}
	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if err := s.MarkUserSendError(ctx, chatID, "403 Forbidden: bot blocked by user"); err != nil {
		t.Fatalf("MarkUserSendError: %v", err)
	}

	var got model.User
	if err := col.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&got); err != nil {
		t.Fatalf("FindOne: %v", err)
	}

	if got.LastSendError != "403 Forbidden: bot blocked by user" {
		t.Errorf("LastSendError = %q, want %q", got.LastSendError, "403 Forbidden: bot blocked by user")
	}
	if time.Since(got.LastSendErrorAt) > 2*time.Second {
		t.Errorf("LastSendErrorAt слишком старый: %v", got.LastSendErrorAt)
	}
	if !got.IsActive {
		t.Error("IsActive = false, want true (обычная ошибка не банит)")
	}
}

// TestMarkUserSendError_BansChatDenied — chat.denied банит.
func TestMarkUserSendError_BansChatDenied(t *testing.T) {
	ctx := context.Background()
	s, col := setupStore(t)
	chatID := "351317206"

	user := &model.User{
		ChatID:   chatID,
		Token:    "tok_" + chatID,
		IsActive: true,
	}
	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	errText := "send message failed: 403 Forbidden: chat.denied"
	if err := s.MarkUserSendError(ctx, chatID, errText); err != nil {
		t.Fatalf("MarkUserSendError: %v", err)
	}

	var got model.User
	if err := col.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&got); err != nil {
		t.Fatalf("FindOne: %v", err)
	}

	if got.LastSendError != errText {
		t.Errorf("LastSendError = %q", got.LastSendError)
	}
	if got.IsActive {
		t.Error("IsActive = true, want false (chat.denied должен банить)")
	}
}

// TestMarkUserSendError_BansDialogSuspended — error.dialog.suspended банит.
func TestMarkUserSendError_BansDialogSuspended(t *testing.T) {
	ctx := context.Background()
	s, col := setupStore(t)
	chatID := "367756978"

	user := &model.User{
		ChatID:   chatID,
		Token:    "tok_" + chatID,
		IsActive: true,
	}
	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	errText := "send message failed: 403 Forbidden: error.dialog.suspended"
	if err := s.MarkUserSendError(ctx, chatID, errText); err != nil {
		t.Fatalf("MarkUserSendError: %v", err)
	}

	var got model.User
	if err := col.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&got); err != nil {
		t.Fatalf("FindOne: %v", err)
	}

	if got.IsActive {
		t.Error("IsActive = true, want false (error.dialog.suspended должен банить)")
	}
}

// TestMarkUserSendError_OtherErrors_NoBan — прочие ошибки не банит.
func TestMarkUserSendError_OtherErrors_NoBan(t *testing.T) {
	ctx := context.Background()
	s, col := setupStore(t)
	chatID := "999888777"

	user := &model.User{
		ChatID:   chatID,
		Token:    "tok_" + chatID,
		IsActive: true,
	}
	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	tests := []struct {
		name    string
		errText string
	}{
		{"429 Too Many Requests", "429 Too Many Requests"},
		{"500 Internal Server Error", "500 Internal Server Error"},
		{"connection refused", "dial tcp: connection refused"},
		{"403 without chat.denied", "403 Forbidden: bot was deleted by user"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := s.MarkUserSendError(ctx, chatID, tc.errText); err != nil {
				t.Fatalf("MarkUserSendError: %v", err)
			}

			var got model.User
			if err := col.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&got); err != nil {
				t.Fatalf("FindOne: %v", err)
			}
			if !got.IsActive {
				t.Errorf("IsActive = false при %q, хотели IsActive = true", tc.errText)
			}
		})
	}
}

// TestMarkUserSendError_GroupChat — работает и по group_chat_id.
func TestMarkUserSendError_GroupChat(t *testing.T) {
	ctx := context.Background()
	s, col := setupStore(t)
	groupChatID := "-1001234567890"

	user := &model.User{
		ChatID:      "000111222",
		GroupChatID: groupChatID,
		Token:       "tok_group",
		IsActive:    true,
	}
	if err := s.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	errText := "403 Forbidden: chat.denied"
	if err := s.MarkUserSendError(ctx, groupChatID, errText); err != nil {
		t.Fatalf("MarkUserSendError: %v", err)
	}

	var got model.User
	if err := col.FindOne(ctx, bson.M{"chat_id": "000111222"}).Decode(&got); err != nil {
		t.Fatalf("FindOne: %v", err)
	}
	if got.IsActive {
		t.Error("IsActive = true, want false (группа с chat.denied)")
	}
}
