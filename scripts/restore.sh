#!/bin/bash
if [ -z "$FILE" ]; then
  echo "Укажите файл бэкапа: make restore FILE=./backups/2mon_20260516_034829.archive"
  echo ""
  echo "Доступные бэкапы:"
  ls -lh ./backups/ | grep -v env
  exit 1
fi

if [ ! -f "$FILE" ]; then
  echo "Файл не найден: $FILE"
  exit 1
fi

echo "Восстанавливаем из $FILE ..."
docker cp "$FILE" 2mon-mongo:/tmp/restore.archive
docker exec 2mon-mongo mongorestore --archive=/tmp/restore.archive --drop
echo "Готово. Перезапускать 2mon не нужно."
