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

В панели MAX укажите URL вебхука: https://ваш-домен.ru/bot

Подробнее в разделе «Как подключить Zabbix» в README.
