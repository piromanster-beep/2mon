#!/bin/bash
docker exec 2mon-mongo mongosh --quiet --eval "EJSON.stringify(db.users.find().toArray())" | python3 -m json.tool
