.PHONY: help test-alert test-webhook backup restore

help:
	@echo "Доступные команды:"
	@echo "  make test-alert              - тестовый алерт из Zabbix (Docker)"
	@echo "  make test-webhook TOKEN=xxx  - тестовый вебхук напрямую"
	@echo "  make backup                  - бэкап MongoDB и .env"
	@echo "  make restore FILE=путь       - восстановить из бэкапа"

test-alert:
	./scripts/test-alert.sh

test-webhook:
	TOKEN=$(TOKEN) ./scripts/test-webhook.sh

backup:
	./scripts/backup.sh

restore:
	FILE=$(FILE) ./scripts/restore.sh
