markdown

# Настройка HTTPS для 2mon

Для работы вебхуков MAX требуется HTTPS на 443 порту.

## 1. Установка Nginx и Certbot

```bash
sudo apt install -y nginx certbot python3-certbot-nginx
```

## 2. Конфиг Nginx

Скопируйте конфиг из репозитория и замените домен:
```bash

sudo cp deploy/nginx-2mon.conf /etc/nginx/sites-available/2mon
sudo sed -i 's/server_name 2mon.ru/server_name ваш-домен.ru/' /etc/nginx/sites-available/2mon
sudo ln -s /etc/nginx/sites-available/2mon /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## 3. Получение SSL-сертификата
```bash

sudo certbot --nginx -d ваш-домен.ru
```
Следуйте инструкциям Certbot (введите email, согласитесь с условиями).

## 4. Проверка
```bash

curl -sI https://ваш-домен.ru | head -5
```
Должен вернуть HTTP/1.1 200 OK.

## 5. Вебхук MAX


Вебхук настраивается через API MAX. Замените `ТОКЕН_БОТА` на токен из панели MAX:

```bash
curl -X POST "https://platform-api.max.ru/subscriptions" \
  -H "Authorization: ТОКЕН_БОТА" \
  -H "Content-Type: application/json" \
  -d '{
  "url": "https://ваш-домен.ru/bot",
  "update_types": ["message_created", "bot_started"]
}'
```

## 6. Домен MAX API и сертификат Минцифры

MAX рекомендует домен `platform-api2.max.ru` вместо `platform-api.max.ru`.
Запросы к `platform-api2` подписаны корневым сертификатом Минцифры, которого
нет в стандартном наборе `ca-certificates` Alpine. Если переключаете
`MAX_API_URL` на `platform-api2.max.ru`, смонтируйте сертификат в контейнер
и добавьте его в доверенные:

```yaml
# docker-compose.yml, сервис app
    volumes:
      - ./certs/russian_trusted_root_ca_pem.crt:/usr/local/share/ca-certificates/russian_trusted_root_ca_pem.crt:ro
```

```bash
# после монтирования обновите хранилище доверенных сертификатов
docker exec 2mon update-ca-certificates
```

Без доверенного корневого сертификата запросы к `platform-api2.max.ru`
падают с ошибкой TLS (`x509: certificate signed by unknown authority`).
Поэтому дефолт в проекте остаётся `platform-api.max.ru`.

Команды бота регистрируются отдельным запросом `PATCH /me/commands`
(делается автоматически при старте сервиса).
