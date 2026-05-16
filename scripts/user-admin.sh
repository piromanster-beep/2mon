#!/bin/bash
# Сделать пользователя админом
# Использование: make user-admin CHAT_ID=284889488
if [ -z "$CHAT_ID" ]; then
  echo "Укажите CHAT_ID: make user-admin CHAT_ID=284889488"
  exit 1
fi
docker exec 2mon-mongo mongosh --quiet --eval "use 2mon; db.users.updateOne({chat_id: '$CHAT_ID'}, {\$set: {is_admin: true}})"
echo "Пользователь $CHAT_ID теперь админ"
