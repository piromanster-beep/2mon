#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups"
S3_BUCKET="s3://2mon-backup"
S3_PREFIX="mongo"

# Дата для имён файлов
DATE=$(date +%Y%m%d_%H%M%S)
MONGO_FILE="2mon_$DATE.archive"
ENV_FILE="env_$DATE"

mkdir -p "$BACKUP_DIR"

# 1. Бэкап MongoDB
echo "Делаем бэкап MongoDB..."
docker exec 2mon-mongo mongodump --archive=/tmp/backup.archive --db=2mon
docker cp 2mon-mongo:/tmp/backup.archive "$BACKUP_DIR/$MONGO_FILE"

# 2. Бэкап .env
echo "Бэкап .env..."
cp .env "$BACKUP_DIR/$ENV_FILE"

# 3. Загрузка в S3
echo "Загружаем в S3..."
s3cmd put "$BACKUP_DIR/$MONGO_FILE" "$S3_BUCKET/$S3_PREFIX/$MONGO_FILE" --no-progress
s3cmd put "$BACKUP_DIR/$ENV_FILE" "$S3_BUCKET/$S3_PREFIX/$ENV_FILE" --no-progress

# 4. Удаляем старые локальные бэкапы (старше 7 дней)
find "$BACKUP_DIR" -mtime +7 -type f -delete

echo ""
echo "Готово!"
echo "  Локально: $BACKUP_DIR/$MONGO_FILE"
echo "  В S3: $S3_BUCKET/$S3_PREFIX/$MONGO_FILE"
echo ""
echo "Восстановить:"
echo "  make restore FILE=$BACKUP_DIR/$MONGO_FILE"
echo ""
echo "Все локальные бэкапы:"
ls -lh "$BACKUP_DIR/" | tail -5
