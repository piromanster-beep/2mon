.PHONY: help fmt vet test check test-alert test-webhook backup restore

help:
	@echo "Доступные команды:"
	@echo "  make check                   - gofmt + go vet + go test"
	@echo "  make test                    - тесты (go test ./...)"
	@echo "  make test-alert              - тестовый алерт из Zabbix (Docker)"
	@echo "  make test-webhook TOKEN=xxx  - тестовый вебхук напрямую"
	@echo "  make backup                  - бэкап MongoDB и .env"
	@echo "  make restore FILE=путь       - восстановить из бэкапа"

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test -count=1 ./...

check: fmt vet test

test-alert:
	./scripts/test-alert.sh

test-webhook:
	TOKEN=$(TOKEN) ./scripts/test-webhook.sh

backup:
	./scripts/backup.sh

restore:
	FILE=$(FILE) ./scripts/restore.sh
