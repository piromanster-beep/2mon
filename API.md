API 2mon

ВЕБХУК ДЛЯ СИСТЕМ МОНИТОРИНГА

POST /wh/{token}

Заголовки:
  Content-Type: application/json

Тело запроса:
  {
    "subject": "Тема сообщения",
    "message": "Текст сообщения",
    "severity": "critical|warning|info"
  }

Ответы:
  200 {"status":"queued"}          — сообщение принято в очередь
  200 {"status":"heartbeat_ok"}    — heartbeat принят (subject=heartbeat)
  404 "invalid token"              — токен не найден
  403 "user disabled"              — пользователь заблокирован
  429 "daily limit exceeded"       — дневной лимит превышен

Severity влияет на эмодзи в сообщении:
  critical, high, disaster -> красный кружок
  warning, average -> жёлтый треугольник
  info -> синий значок


ВЕБХУК ДЛЯ MAX (БОТ)

POST /bot

Принимает обновления от MAX (webhook).
Формат: JSON согласно MAX Bot API.

Поддерживаемые update_type:
  message_created — текстовое сообщение
  bot_started — первое нажатие "Начать"

Команды бота:
  /start — регистрация, получение токена
  /token — показать токен
  /status — статистика за сегодня
  /heartbeat — статус heartbeat
  /help — справка


АДМИНКА

GET /admin — страница входа
POST /admin/login — вход (password)
GET /admin/dashboard — список пользователей
POST /admin/ban/{chat_id} — заблокировать/разблокировать
POST /admin/limit/{chat_id} — изменить дневной лимит


HEARTBEAT

Для мониторинга доступности Zabbix:
  Zabbix отправляет вебхук с subject=heartbeat раз в 5 минут
  2mon обновляет last_heartbeat
  Если heartbeat не приходит 15 минут:
    — админу: уведомление в MAX
    — пользователю: уведомление в MAX
  При восстановлении heartbeat:
    — пользователю: "Heartbeat восстановлен"
