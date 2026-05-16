#!/bin/bash
# Отправляет тестовый алерт из Zabbix в MAX
# Использование: make test-alert
docker exec zabbix-test-zabbix-server-1 zabbix_sender -z 127.0.0.1 -s "Zabbix server" -k test.trigger -o 1
docker exec zabbix-test-zabbix-server-1 zabbix_sender -z 127.0.0.1 -s "Zabbix server" -k test.trigger -o 0
