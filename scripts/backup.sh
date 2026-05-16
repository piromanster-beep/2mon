#!/bin/bash
BACKUP_DIR="./backups"
mkdir -p "$BACKUP_DIR"
DATE=$(date +%Y%m%d_%H%M%S)
FILE="$BACKUP_DIR/2mon_$DATE.archive"

echo "Делаем бэкап MongoDB..."
docker exec 2mon-mongo mongodump --archive=/tmp/backup.archive --db=2mon
docker cp 2mon-mongo:/tmp/backup.archive "$FILE"

echo "Бэкап .env..."
cp .env "$BACKUP_DIR/env_$DATE"

echo ""
echo "Готово: $FILE"
echo ""
echo "Чтобы восстановить:"
echo "  make restore FILE=$FILE"
echo ""
echo "Все бэкапы:"
ls -lh "$BACKUP_DIR/"
