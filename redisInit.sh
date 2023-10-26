#!/bin/bash

# docker pull redis
docker run -d --name owo-redis --network oj_network \
    -p 6379:6379 --shm-size=128m --restart=always \
    -v /usr/local/docker/lazyliqiquan/data/redis:/data \
    redis
