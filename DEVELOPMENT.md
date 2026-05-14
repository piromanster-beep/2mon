РАЗРАБОТКА 2mon

БЫСТРЫЙ СТАРТ

Требования:
  - Go 1.21+
  - Docker и Docker Compose
  - Токен бота MAX (получить в панели MAX для партнёров)
  - Домен с HTTPS (можно localhost для разработки)

Локальный запуск:
  git clone https://gitflic.ru/piroman99/2mon.git
  cd 2mon
  cp .env.example .env
  # заполнить .env
  docker compose up -d

Для разработки без Docker:
  Нужен локальный MongoDB на localhost:27017
  В .env указать MONGO_URI=mongodb://localhost:27017
  go mod tidy
  go run ./cmd/server/


СТРУКТУРА ПРОЕКТА

cmd/server/main.go          — точка входа
internal/
  config/config.go          — загрузка .env
  model/model.go            — структуры данных
  store/mongo.go            — работа с MongoDB
  sender/sender.go          — очередь + rate limiter + MAX API
  handler/
    webhook.go              — приём вебхуков от Zabbix
    bot.go                  — обработка команд бота MAX
    admin.go                — админка
  notifier/notifier.go      — уведомления админам
  scheduler/scheduler.go    — cron-задачи
web/admin.html              — HTML админки
deploy/
  nginx-2mon.conf           — конфиг Nginx
  HTTPS.md                  — инструкция по HTTPS


КАК ДОБАВИТЬ НОВЫЙ СЕРВИС МОНИТОРИНГА

2mon принимает любые вебхуки с JSON:
  {"subject":"...", "message":"...", "severity":"..."}

Для Prometheus:
  Использовать Alertmanager webhook receiver
  URL: https://2mon.ru/wh/TOKEN

Для Grafana:
  Создать Contact point типа webhook
  URL: https://2mon.ru/wh/TOKEN
  В Optional webhook settings включить "Use PUT method"

Для самописных скриптов:
  curl -X POST https://2mon.ru/wh/TOKEN \
    -H "Content-Type: application/json" \
    -d '{"subject":"Бэкап","message":"Успешно","severity":"info"}'

Если формат вебхуха отличается от стандартного,
можно добавить адаптер в handler/webhook.go.


ЗАВИСИМОСТИ

  github.com/go-chi/chi/v5       — HTTP-роутер
  go.mongodb.org/mongo-driver    — MongoDB-драйвер
  golang.org/x/time/rate         — rate limiter
  github.com/robfig/cron/v3      — планировщик
  github.com/google/uuid         — генерация токенов
