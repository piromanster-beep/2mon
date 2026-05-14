АРХИТЕКТУРА 2mon

КОМПОНЕНТЫ

1. HTTP-сервер (Go + chi)
   - Принимает вебхуки от систем мониторинга на /wh/{token}
   - Принимает обновления от MAX на /bot
   - Отдаёт админку на /admin

2. Sender (очередь + rate limiter)
   - Все исходящие сообщения в MAX проходят через буферизированный канал
   - Rate limiter ограничивает отправку до 30 сообщений/сек (лимит MAX API)
   - При остановке сервиса очередь дренируется (оставшиеся сообщения отправляются)

3. Scheduler (планировщик)
   - Ежедневная статистика пользователям (9:00 по будням)
   - Ежедневная статистика админам (9:05 по будням)
   - Проверка heartbeat каждую минуту

4. Notifier
   - Уведомления админам о новых регистрациях
   - Уведомления о проблемах с heartbeat

5. MongoDB
   - Коллекция users: chat_id, token, лимиты, heartbeat
   - Коллекция messages_log: история сообщений для статистики

ПОТОКИ ДАННЫХ

Вебхук от Zabbix:
  Zabbix -> POST /wh/{token} -> проверка токена -> проверка лимита
  -> очередь sender -> rate limiter -> MAX API -> пользователю в чат

Команда бота:
  MAX -> POST /bot -> парсинг update_type -> обработка команды
  -> ответ через sender -> MAX API -> пользователю

Heartbeat:
  Zabbix Web scenario -> POST /wh/{token} (subject=heartbeat)
  -> обновление last_heartbeat -> OK
  Scheduler раз в минуту проверяет просроченные heartbeat
  -> если >15 минут -> alert админу и пользователю

БЕЗОПАСНОСТЬ

- Токен пользователя: UUID v4 (2^122 комбинаций)
- Rate limit на вебхуки: 30 запросов/мин (Nginx)
- Rate limit на админку: 10 запросов/мин (Nginx)
- Порт 8080 привязан к localhost, снаружи не доступен
- MongoDB без пароля, но только внутри Docker-сети
- HTTPS через Let's Encrypt

МОДЕЛИ

User:
  ID, ChatID, Token, IsActive, IsAdmin
  DailyLimit, MsgCountToday, MsgDate
  LastHeartbeat, HeartbeatInterval, HeartbeatAlertSent
  CreatedAt

MessageLog:
  ID, UserID, Source, Status, CreatedAt

WebhookPayload:
  Subject, Message, Severity
