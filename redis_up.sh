#!/bin/bash

CONTAINER_NAME="eraya-redis"
HOST_PORT="6379"
REDIS_PORT="6379"

if [ "$(docker ps -q -f name=$CONTAINER_NAME)" ]; then
    echo "Redis container is already running."
elif [ "$(docker ps -aq -f name=$CONTAINER_NAME)" ]; then
    echo "Starting existing Redis container..."
    docker start $CONTAINER_NAME
else
    echo "Creating and starting new Redis container..."
    docker run --name $CONTAINER_NAME -p $HOST_PORT:$REDIS_PORT -d redis:latest
fi

echo "Redis is up and running on $HOST_PORT!"
