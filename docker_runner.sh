#!/bin/sh
if [ ! -f forumdb.db ]; then
        echo "forumdb.db not found, creating new database..."
        touch forumdb.db
fi

chmod 777 forumdb.db

docker run -it --rm \
        -v $(pwd)/forumdb.db:/app/forumdb.db \
        -e DB_PATH=./forumdb.db \
        -p 8080:8080 \
        forum:v1