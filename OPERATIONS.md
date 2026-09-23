# Эксплуатация 2mon

Документ для тех, кто разворачивает и сопровождает свой экземпляр 2mon.
Устройство сервиса — в [ARCHITECTURE.md](ARCHITECTURE.md), разработка — в
[DEVELOPMENT.md](DEVELOPMENT.md).

## Что где лежит

| Что | Где |
|-----|-----|
| Приложение | контейнер `2mon` (docker compose, сервис `app`) |
| MongoDB | контейнер `2mon-mongo` (образ `mongo:7.0`, том `mongo_data`) |
| Настройки | `.env` (см. `.env.example` и таблицу в [README.md](README.md)) |
| Логи | `docker compose logs` (драйвер json-file, 5 МБ × 3 файла) |
| Nginx | [deploy/nginx-2mon.conf](deploy/nginx-2mon.conf) |
| Бэкап / восстановление | [scripts/backup.sh](scripts/backup.sh), [scripts/restore.sh](scripts/restore.sh) |

## Обновление

```bash
cd /path/to/2mon
git pull
docker compose build app
docker compose up -d
docker compose logs -f app
```

- Образ собирается до перезапуска контейнера, поэтому простоя почти нет.
- При остановке сервис завершается корректно: ждёт до 15 секунд in-flight
  запросы и дренирует очередь sender. Не убивайте контейнер через
  `docker kill` без нужды.
- Схема MongoDB приводится к нужному виду автоматически при старте (см.
  раздел «Миграции схемы»).
- Изменения в `deploy/nginx-2mon.conf` применяются отдельно:
  `sudo nginx -t && sudo systemctl reload nginx`.

### Откат

```bash
git checkout <прошлый-коммит-или-тег>
docker compose build app
docker compose up -d
```

Данные в MongoDB при откате кода не меняются — схема только дополняется.
Откат обратно на более старую версию, которая не знает про TTL-индекс, не
ломает базу: лишний индекс выглядит как обычный.

## Миграции схемы

Отдельного шага «применить миграции» нет: сервис приводит MongoDB к нужному
виду сам при старте.

- **Индексы** создаются и обновляются автоматически при запуске
  (`store.createIndexes`). Повторный запуск безопасен.
- **Ретенция журнала.** Параметр `LOG_RETENTION_DAYS`:
  - `> 0` — MongoDB сама удаляет старые записи `messages_log` через
    TTL-индекс на `created_at` (дефолт — 180 дней);
  - `<= 0` — автоочистка выключается, остаётся обычный индекс.
  Смена значения применяется при следующем старте: старый индекс на
  `created_at` удаляется и создаётся заново с нужными опциями.
- **Нет прав на `dropIndex`?** Сервис не падает: пишет в лог строку вида
  `[store] retention: ...` и продолжает работу без автоочистки. Дайте
  пользователю MongoDB права на изменение коллекции `messages_log`, если
  ретенция нужна.

Чтобы применить изменение `LOG_RETENTION_DAYS`, измените `.env` и
перезапустите приложение:

```bash
docker compose up -d --force-recreate app
```

## Бэкап и восстановление

### Бэкап

```bash
make backup        # или: ./scripts/backup.sh
```

Скрипт кладёт в `./backups` дамп MongoDB (`mongodump --archive --db=2mon`) и
копию `.env`. Бэкапить нужно **и то, и другое**: восстановленный дамп без
актуального `.env` бесполезен — там токен бота, пароль админки и строка
подключения к базе.

Ручной вариант без Makefile:

```bash
docker exec 2mon-mongo mongodump --archive=/tmp/backup.archive --db=2mon
docker cp 2mon-mongo:/tmp/backup.archive ./backups/2mon_$(date +%Y%m%d).archive
cp .env ./backups/env_$(date +%Y%m%d)
```

Регулярный бэкап (пример cron на сервере):

```cron
0 4 * * * cd /path/to/2mon && ./scripts/backup.sh >> backups/backup.log 2>&1
```

### Восстановление

```bash
make restore FILE=./backups/2mon_20260516_034829.archive
```

Скрипт копирует дамп в контейнер и делает `mongorestore --drop` — текущая
база перезаписывается. Перезапускать приложение не нужно.

> `--drop` удаляет коллекции, которые есть в дампе, перед вставкой.
> Перед восстановлением на проде сделайте свежий бэкап текущего состояния.

### Перенос на другой сервер

1. Скопируйте `backups/<дамп>` и соответствующий `backups/env_<дата>`.
2. Разверните 2mon на новом сервере (см. [README.md](README.md), «Быстрый старт»),
   положите `.env`.
3. `make restore FILE=<дамп>`.

## Healthcheck

`GET /healthz` возвращает `{"status":"ok"}` — процесс жив и HTTP отвечает.
MongoDB в этой проверке намеренно не пингуется, чтобы healthcheck не шумел
при кратковременных сбоях базы.

```bash
curl -fsS http://127.0.0.1:8080/healthz
```

Порт привязан к `127.0.0.1` (см. `docker-compose.yml`), поэтому извне
`/healthz` недоступен: `deploy/nginx-2mon.conf` проксирует только `/bot` и
`/wh/`. Для внешнего контроля доступности либо добавьте `location = /healthz`
в Nginx, либо используйте сторонний аптайм-монитор по реальной точке входа.

### Runbook: `/healthz` не отвечает

1. `docker compose ps` — контейнер `2mon` должен быть `Up`.
2. `docker compose logs --tail=100 app` — смотрите последние строки
   (ошибки конфига, подключения к Mongo — они фатальны на старте).
3. Частые причины: `.env` не заполнен или отсутствует обязательная переменная
   (`MONGO_URI`, `MAX_BOT_TOKEN`, `ADMIN_PASSWORD`); Mongo не поднялась;
   порт 8080 занят.
4. `docker compose up -d` для перезапуска. Если проблема в Mongo — проверьте
   `docker compose logs mongo` и том `mongo_data`.

## Схема MongoDB

База `2mon`.

### Коллекция `users`

| Поле | Тип | Смысл |
|------|-----|-------|
| `_id` | string | ObjectID в hex |
| `chat_id` | string | ID личного чата в MAX |
| `token` | string | секрет для `/wh/{token}` (UUID v4) |
| `is_active` | bool | блокировка пользователя |
| `is_admin` | bool | получатель админских уведомлений |
| `daily_limit` | int | дневной лимит сообщений |
| `msg_count_today` | int | счётчик за текущий день |
| `msg_date` | string | дата счётчика, `YYYY-MM-DD` |
| `created_at` | date | регистрация |
| `last_heartbeat` | date | последний heartbeat |
| `heartbeat_interval` | int | ожидаемый интервал, мин |
| `heartbeat_alert_sent` | bool | уведомление об обрыве уже отправлено |
| `group_chat_id` | string | привязанная группа (если есть) |
| `last_send_error` | string | причина последней неудачной отправки в MAX |
| `last_send_error_at` | date | когда ошибка зафиксирована |

Индексы: `token` (уникальный), `chat_id`.

Первого администратора создают вручную:

```javascript
db.users.updateOne({chat_id: "ВАШ_CHAT_ID"}, {$set: {is_admin: true}})
```

### Коллекция `messages_log`

| Поле | Тип | Смысл |
|------|-----|-------|
| `_id` | string | ObjectID в hex |
| `user_id` | string | `users._id` |
| `source` | string | источник (`zabbix`, ...) |
| `status` | string | `success` или `limit_exceeded` |
| `created_at` | date | время записи |

Индексы: `(user_id, created_at)` — для статистики; `created_at` — TTL
(при `LOG_RETENTION_DAYS > 0`) либо обычный.

Полезные запросы:

```javascript
db.messages_log.countDocuments()          // объём журнала
db.messages_log.getIndexes()              // индексы и TTL
```

## Логи

- `docker compose logs -f app` — приложение.
- `docker compose logs -f mongo` — база.
- Драйвер `json-file` ограничен 5 МБ × 3 файла на контейнер, поэтому диск не
  забьётся. Для долгого хранения настройте сбор логов вовне.

Типовые префиксы сообщений приложения: `[webhook]`, `[bot]`, `[sender]`,
`[scheduler]`, `[store]`, `[commands]`.

## Диагностика

**Вебхук отвечает 404 `invalid token`**
Токен поменяли (`/newtoken confirm`) или скопировали с ошибкой. Актуальный
токен всегда можно получить командой `/token` в личке с ботом.

**403 `user disabled`**
Пользователь заблокирован в админке (`is_active: false`).

**429 `daily limit exceeded`**
Исчерпан дневной лимит. Счётчик сбрасывается в полночь по времени сервера;
лимит можно поднять в админке.

**429 `queue full`**
Переполнена очередь отправки (`QUEUE_SIZE`). Обычно значит, что MAX отвечает
медленно или недоступен — проверьте `docker compose logs app` на строки
`[sender]`.

**Сообщения не приходят в MAX**
В админке для пользователя видна причина последней неудачной отправки
(`last_send_error`), например `HTTP 403`. Частые случаи: пользователь
заблокировал бота; бота не назначили администратором группы (иначе MAX не
отдаёт сообщения из группы).

**Heartbeat не приходит**
Статус покажет `/heartbeat` в боте. Проверьте Web scenario в Zabbix
(см. [ZABBIX.md](ZABBIX.md)): URL, метод, заголовок
`Content-Type: application/json`, тело с `subject: heartbeat`. Уведомление об
обрыве уходит, если heartbeat молчит больше 15 минут.

**В логах `[commands] ...` при старте**
Не удалось обновить подсказки команд в MAX (`PATCH /me/commands`). Не
фатально: уже настроенные команды работают. Проверьте токен бота и
доступность `MAX_API_URL`.

**TLS `x509: certificate signed by unknown authority`**
Обращаетесь к `platform-api2.max.ru` без корневого сертификата Минцифры. См.
[deploy/HTTPS.md](deploy/HTTPS.md) или вернитесь на `platform-api.max.ru`.

**MongoDB не стартует / сервис ждёт базу**
Проверьте `docker compose logs mongo`, свободное место и права на том
`mongo_data`. Контейнеры объявлены с `restart: always`, так что после
перезагрузки сервис поднимется сам — проконтролируйте через `/healthz`.

## Мониторинг самого 2mon

- Вебхук-канал `/wh/{token}` — сюда шлют системы мониторинга; heartbeat-контур
  сообщает об обрыве, если с той стороны что-то отвалилось.
- Для внешнего контроля доступности самого 2mon используйте `/healthz`
  (см. выше) или сторонний аптайм-монитор по вашей точке входа. Подключение
  систем мониторинга — в [MONITORING.md](MONITORING.md).
