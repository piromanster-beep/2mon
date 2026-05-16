#!/bin/bash
# Забанить/разбанить пользователя
# Использование: make user-ban CHAT_ID=284889488
if [ -z "$CHAT_ID" ]; then
  echo "Укажите CHAT_ID: make user-ban CHAT_ID=284889488"
  exit 1
fi
STATUS=$(docker exec 2mon-mongo mongosh --quiet --eval "use 2mon; db.users.findOne({chat_id: '$CHAT_ID'}).is_active")
if [ "$STATUS" = "true" ]; then
  docker exec 2mon-mongo mongosh --quiet --eval "use 2mon; db.users.updateOne({chat_id: '$CHAT_ID'}, {\$set: {is_active: false}})"
  echo "Пользователь $CHAT_ID забанен"
else
  docker exec 2mon-mongo mongosh --quiet --eval "use 2mon; db.users.updateOne({chat_id: '$CHAT_ID'}, {\$set: {is_active: true}})"
  echo "Пользователь $CHAT_ID разбанен"
fi
