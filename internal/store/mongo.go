package store

import (
	"context"
	"fmt"
	"time"

	"gitflic.ru/piroman99/2mon/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store — хранилище данных
type Store struct {
	client *mongo.Client
	db     *mongo.Database
	users  *mongo.Collection
	logs   *mongo.Collection
}

// New — подключение к MongoDB
func New(ctx context.Context, uri string) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	db := client.Database("2mon")

	s := &Store{
		client: client,
		db:     db,
		users:  db.Collection("users"),
		logs:   db.Collection("messages_log"),
	}

	// Создаём индексы
	if err := s.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("create indexes: %w", err)
	}

	return s, nil
}

// createIndexes — индексы для быстрого поиска
func (s *Store) createIndexes(ctx context.Context) error {
	// Уникальный индекс на токен
	_, err := s.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Индекс на chat_id
	_, err = s.users.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "chat_id", Value: 1}},
	})
	if err != nil {
		return err
	}

	// Индекс на created_at для статистики
	_, err = s.logs.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	})
	if err != nil {
		return err
	}

	return nil
}

// Close — закрытие соединения
func (s *Store) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// ============= ПОЛЬЗОВАТЕЛИ =============

// FindByToken — найти пользователя по токену
func (s *Store) FindByToken(ctx context.Context, token string) (*model.User, error) {
	var user model.User
	err := s.users.FindOne(ctx, bson.M{"token": token}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil // не нашли — не ошибка
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByChatID — найти пользователя по chat_id
func (s *Store) FindByChatID(ctx context.Context, chatID string) (*model.User, error) {
	var user model.User
	err := s.users.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser — создать нового пользователя
func (s *Store) CreateUser(ctx context.Context, user *model.User) error {
	user.CreatedAt = time.Now()
	user.MsgDate = time.Now().Format("2006-01-02")
	_, err := s.users.InsertOne(ctx, user)
	return err
}

// UpdateUser — обновить пользователя
func (s *Store) UpdateUser(ctx context.Context, user *model.User) error {
	_, err := s.users.ReplaceOne(ctx, bson.M{"_id": user.ID}, user)
	return err
}

// IncrementMsgCount — атомарно увеличить счётчик сообщений за сегодня
// Возвращает: новое значение счётчика, ошибку
func (s *Store) IncrementMsgCount(ctx context.Context, user *model.User) (int, error) {
	today := time.Now().Format("2006-01-02")

	// Если дата сменилась — сбрасываем счётчик в 0
	if user.MsgDate != today {
		_, err := s.users.UpdateOne(
			ctx,
			bson.M{"_id": user.ID},
			bson.M{
				"$set": bson.M{
					"msg_date":        today,
					"msg_count_today": 1,
				},
			},
		)
		if err != nil {
			return 0, err
		}
		return 1, nil
	}

	// Иначе увеличиваем на 1
	result := s.users.FindOneAndUpdate(
		ctx,
		bson.M{"_id": user.ID},
		bson.M{"$inc": bson.M{"msg_count_today": 1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updated model.User
	if err := result.Decode(&updated); err != nil {
		return 0, err
	}

	return updated.MsgCountToday, nil
}

// FindAdmins — найти всех администраторов
func (s *Store) FindAdmins(ctx context.Context) ([]model.User, error) {
	cursor, err := s.users.Find(ctx, bson.M{"is_admin": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var admins []model.User
	if err := cursor.All(ctx, &admins); err != nil {
		return nil, err
	}
	return admins, nil
}

// FindActiveUsers — найти всех активных пользователей
func (s *Store) FindActiveUsers(ctx context.Context) ([]model.User, error) {
	cursor, err := s.users.Find(ctx, bson.M{"is_active": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []model.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// ============= ЛОГИ СООБЩЕНИЙ =============

// LogMessage — записать сообщение в лог
func (s *Store) LogMessage(ctx context.Context, log *model.MessageLog) error {
	log.CreatedAt = time.Now()
	_, err := s.logs.InsertOne(ctx, log)
	return err
}

// CountMessagesByDate — количество сообщений за дату
func (s *Store) CountMessagesByDate(ctx context.Context, date string) (int, error) {
	start, _ := time.Parse("2006-01-02", date)
	end := start.Add(24 * time.Hour)

	filter := bson.M{
		"created_at": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	count, err := s.logs.CountDocuments(ctx, filter)
	return int(count), err
}

// CountUsersByDate — количество новых пользователей за дату
func (s *Store) CountUsersByDate(ctx context.Context, date string) (int, error) {
	start, _ := time.Parse("2006-01-02", date)
	end := start.Add(24 * time.Hour)

	filter := bson.M{
		"created_at": bson.M{
			"$gte": start,
			"$lt":  end,
		},
	}

	count, err := s.users.CountDocuments(ctx, filter)
	return int(count), err
}

// TopUsersByMessages — топ пользователей по количеству сообщений за дату
func (s *Store) TopUsersByMessages(ctx context.Context, date string, limit int) ([]model.UserStat, error) {
	start, _ := time.Parse("2006-01-02", date)
	end := start.Add(24 * time.Hour)

	pipeline := mongo.Pipeline{
		// Фильтр по дате
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{
				"$gte": start,
				"$lt":  end,
			},
		}}},
		// Группируем по user_id, считаем количество
		{{Key: "$group", Value: bson.M{
			"_id":   "$user_id",
			"count": bson.M{"$sum": 1},
		}}},
		// Сортируем по убыванию
		{{Key: "$sort", Value: bson.M{"count": -1}}},
		// Ограничиваем количество
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := s.logs.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stats []model.UserStat
	for cursor.Next(ctx) {
		var result struct {
			UserID string `bson:"_id"`
			Count  int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		stats = append(stats, model.UserStat{
			ChatID:   result.UserID,
			MsgCount: result.Count,
		})
	}

	return stats, nil
}
