.PHONY: help fmt vet test check test-alert test-webhook backup restore

help:
	@echo "Доступные команды:"
	@echo "  make check                   - gofmt + go vet + go test"
	@echo "  make test                    - тесты (go test ./...)"
	@echo "  make test-alert              - тестовый алерт из Zabbix (Docker)"
	@echo "  make test-webhook TOKEN=xxx  - тестовый вебхук напрямую (curl)"
	@echo "  make backup                  - бэкап MongoDB и .env"
	@echo "  make restore FILE=путь       - восстановить из бэкапа"
	@echo "  make restore-from-s3 FILE=путь       - восстановить из S3 бэкапа"

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
	@test -n "$(TOKEN)" || { echo "Укажите токен: make test-webhook TOKEN=xxx"; exit 1; }
	curl -fsS -X POST "http://127.0.0.1:8080/wh/$(TOKEN)" \
		-H "Content-Type: application/json" \
		-d '{"subject":"Тест","message":"Проверка вебхука","severity":"info"}'

backup:
	./scripts/backup.sh

restore:
	FILE=$(FILE) ./scripts/restore.sh

restore-from-s3:
	FILE=$(FILE) ./scripts/restore.sh
