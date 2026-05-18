## НАСТРОЙКА ZABBIX ДЛЯ 2mon

Два компонента:
1. Webhook для алертов — Zabbix шлёт уведомления о проблемах
2. Web scenario для heartbeat — Zabbix шлёт сигнал "я жив"


1. WEBHOOK ДЛЯ АЛЕРТОВ

1.1. Media type
  Alerts -> Media types -> Create media type
  Type: Webhook
  Name: 2mon
  Parameters:
    URL: https://ваш-сервер/wh/{ALERT.SENDTO}
    Subject: {ALERT.SUBJECT}
    Message: {ALERT.MESSAGE}
    Severity: {ALERT.SEVERITY}
    
  Script:
  ```
    try {
        var params = JSON.parse(value);
        var body = JSON.stringify({
            subject: params.Subject,
            message: params.Message,
            severity: params.Severity
        });
        var request = new HttpRequest();
        request.addHeader('Content-Type: application/json');
        var response = request.post(params.URL, body);
        if (request.getStatus() !== 200) {
            throw 'HTTP ' + request.getStatus() + ': ' + response;
        }
        return 'OK';
    }
    catch (err) {
        throw err;
    }
```
## Настройка эмодзи в сообщениях MAX

По умолчанию Zabbix отправляет в MAX уведомления без эмодзи. Чтобы добавить визуальные индикаторы (❌, ✅, ℹ️), нужно настроить **Message templates** в Media Type.

### Шаги

1. `Administration` → `Media types` → выберите ваш Media Type (например, `MAX`)
2. Перейдите на вкладку **Message templates**
3. Добавьте три шаблона:

#### Problem

| Поле | Значение |
|------|----------|
| Message type | `Problem` |
| Subject | `❌ Problem: {EVENT.NAME}` |
| Message | `❌ Проблема началась в {EVENT.TIME} {EVENT.DATE}\nПроблема: {EVENT.NAME}\nХост: {HOST.NAME}\nВажность: {EVENT.SEVERITY}\nID проблемы: {EVENT.ID}` |

#### Problem recovery

| Поле | Значение |
|------|----------|
| Message type | `Problem recovery` |
| Subject | `✅ Resolved in {EVENT.DURATION}: {EVENT.NAME}` |
| Message | `✅ Проблема решена в {EVENT.RECOVERY.TIME} {EVENT.RECOVERY.DATE}\nПроблема: {EVENT.RECOVERY.NAME}\nДлительность: {EVENT.DURATION}\nХост: {HOST.NAME}\nВажность: {EVENT.SEVERITY}` |

#### Problem update (опционально)

| Поле | Значение |
|------|----------|
| Message type | `Problem update` |
| Subject | `ℹ️ Updated: {EVENT.NAME}` |
| Message | `{USER.FULLNAME} {EVENT.UPDATE.ACTION} проблему.\n{EVENT.UPDATE.MESSAGE}` |

4. Сохраните Media Type (кнопка **Update**)

### Проверка

- Проблема → сообщение начинается с ❌
- Восстановление → сообщение начинается с ✅
- Комментарий → сообщение начинается с ℹ️

### Примечание

Эмодзи — это обычные символы Юникода. MAX поддерживает их без дополнительных настроек.
1.2. Пользователь
  Administration -> Users -> ваш пользователь -> Media -> Add
  Type: 2mon
  Send to: ваш_токен_из_бота

1.3. Action
  Configuration -> Actions -> Trigger actions -> Create action
  Conditions: Trigger severity >= Warning
  Operations: Send message to users via 2mon


2.  WEB SCENARIO ДЛЯ HEARTBEAT

2.1. Web scenario
  Configuration -> Hosts -> Zabbix server -> Web scenarios -> Create web scenario
  Name: Heartbeat to 2mon
  Update interval: 5m
  Steps -> Add:
    Name: Heartbeat
    URL: https://ваш-сервер/wh/ВАШ_ТОКЕН
    Post type: Raw data
    Raw data: {"subject":"heartbeat","message":"ping","severity":"info"}
    Headers: Name: Content-Type, Value: application/json
    Required status codes: 200

2.2. Проверка
  В боте MAX команда /heartbeat
  Должен показать: Heartbeat: OK, Последний сигнал: Xs назад

2.3. Отказоустойчивость
  Если heartbeat не приходит 15 минут:
    — Админ получает уведомление в MAX
    — Пользователь получает уведомление в MAX
  При восстановлении heartbeat пользователь получает:
    "Heartbeat восстановлен. Ваш Zabbix снова на связи."
