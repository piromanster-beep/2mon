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

## Сертификаты Минцифры

Для работы с `platform-api2.max.ru` в Docker-образ добавлены корневой и выпускающий сертификаты Минцифры (папка `certs/`).

Сертификаты периодически обновляются. Актуальные версии — на портале Госуслуг: https://www.gosuslugi.ru/crt

При обновлении: замените файлы в `certs/`, пересоберите образ `docker compose up -d --build`.


Команды бота регистрируются отдельным запросом `PATCH /me/commands`
(делается автоматически при старте сервиса).
