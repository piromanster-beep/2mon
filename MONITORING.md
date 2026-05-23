ПОДКЛЮЧЕНИЕ СИСТЕМ МОНИТОРИНГА К 2mon

2mon принимает вебхуки в формате JSON с полями subject, message, severity.
URL вебхука: https://2mon.ru/wh/ВАШ_ТОКЕН


ZABBIX
Подробная инструкция: [ZABBIX.md](ZABBIX.md)


PROMETHEUS + ALERTMANAGER
В Alertmanager добавьте webhook-получатель в alertmanager.yml:

```
receivers:
  - name: '2mon'
    webhook_configs:
      - url: 'https://2mon.ru/wh/ТОКЕН'
        send_resolved: true
        max_alerts: 1
```

Для преобразования формата Prometheus в формат 2mon используйте
шаблоны Alertmanager (custom_fields) или внешний конвертер.


UPTIME KUMA
1. Настройки → Уведомления → Добавить → Webhook
2. URL: https://2mon.ru/wh/ТОКЕН
3. Content-Type: application/json
4. Custom Body:
`   {"subject":"{{monitorJSON['name']}}","message":"{{msg}}","severity":"warning"}`


NETDATA
В файле health_alarm_notify.conf пропишите:
```
SEND_CUSTOM="YES"
CUSTOM_WEBHOOK_URL="https://2mon.ru/wh/ТОКЕН"
CUSTOM_WEBHOOK_MESSAGE='{"subject":"${alarm}","message":"${status_message}","severity":"warning"}'
```

HEARTBEAT ДЛЯ ДРУГИХ СИСТЕМ
В Zabbix используется Web scenario (см. [ZABBIX.md](ZABBIX.md)).
Для других систем — cron на сервере мониторинга:

```
*/5 * * * * curl -s -X POST "https://2mon.ru/wh/ТОКЕН" -H "Content-Type: application/json" -d '{"subject":"heartbeat","message":"ping","severity":"info"}'
```

В ПЛАНАХ
- Grafana OnCall (Formatted Webhook)
- Автоматический конвертер форматов Prometheus/Alertmanager
- Готовые шаблоны для популярных систем
