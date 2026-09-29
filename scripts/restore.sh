#!/bin/bash

BACKUP_DIR="./backups"
S3_BUCKET="s3://2mon-backup/mongo"

if [ -z "$FILE" ]; then
    echo "Укажите файл бэкапа: make restore FILE=./backups/2mon_20260929_211149.archive"
    echo "Или из S3: make restore-from-s3 FILE=2mon_20260929_211149.archive"
    echo ""
    echo "Доступные локальные бэкапы:"
    ls -lh "$BACKUP_DIR/" 2>/dev/null | grep -v env | tail -5
    echo ""
    echo "Доступные в S3:"
    s3cmd ls "$S3_BUCKET/" 2>/dev/null | grep "\.archive" | tail -5
    exit 1
fi

# Если файла нет по указанному пути — ищем в backups/
if [ ! -f "$FILE" ] && [ -f "$BACKUP_DIR/$(basename $FILE)" ]; then
    FILE="$BACKUP_DIR/$(basename $FILE)"
    echo "Найден локально: $FILE"
fi

# Если всё ещё нет — качаем из S3
if [ ! -f "$FILE" ]; then
    BASENAME=$(basename "$FILE")
    echo "Файл не найден локально, качаем из S3: $S3_BUCKET/$BASENAME"
    s3cmd get "$S3_BUCKET/$BASENAME" "$BACKUP_DIR/$BASENAME" --force --no-progress
    if [ $? -ne 0 ]; then
        echo "Не удалось скачать из S3"
        exit 1
    fi
    FILE="$BACKUP_DIR/$BASENAME"
fi

echo "Восстанавливаем из $FILE ..."
docker cp "$FILE" 2mon-mongo:/tmp/restore.archive
docker exec 2mon-mongo mongorestore --archive=/tmp/restore.archive --drop
echo "Готово. Перезапускать 2mon не нужно."
