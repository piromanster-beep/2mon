package model

import "time"

type User struct {
	ID            string    `bson:"_id,omitempty" json:"id"`
	ChatID        string    `bson:"chat_id" json:"chat_id"`
	Token         string    `bson:"token" json:"token"`
	IsActive      bool      `bson:"is_active" json:"is_active"`
	IsAdmin       bool      `bson:"is_admin" json:"is_admin"`
	DailyLimit    int       `bson:"daily_limit" json:"daily_limit"`
	MsgCountToday int       `bson:"msg_count_today" json:"msg_count_today"`
	MsgDate       string    `bson:"msg_date" json:"msg_date"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
}

type MessageLog struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	Source    string    `bson:"source" json:"source"`
	Status    string    `bson:"status" json:"status"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type Message struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

type WebhookPayload struct {
	Subject  string `json:"subject"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type DailyStats struct {
	Date          string     `json:"date"`
	TotalMessages int        `json:"total_messages"`
	NewUsers      int        `json:"new_users"`
	TopUsers      []UserStat `json:"top_users"`
}

type UserStat struct {
	ChatID   string `json:"chat_id"`
	MsgCount int    `json:"msg_count"`
}
