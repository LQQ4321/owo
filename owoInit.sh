#!/bin/bash

# 使用该项目的流程：
# 1.    将该项目下载到服务器
# 2.    给owoInit.sh脚本root权限，然后运行它即可
# 3.    当然，还可以添加一些交互，如获取mysql原始密码等操作

# network
# docker network create -d bridge oj_network

# mysql
# docker pull mysql:8.0
# docker run -d --restart=always --name mysql --network oj_network \
#   -e MYSQL_ROOT_PASSWORD=3515063609563648226 \
#   -v /usr/local/docker/lazyliqiquan/data/mysql:/var/lib/mysql \
#   mysql:8.0

# judger
# docker build -t go-judge -f Dockerfile.judger .
# docker run -d -e ES_DIR=/dev/shm/files/share_judger --privileged \
#   --shm-size=256m --restart=always --name judger --network oj_network \
#   -v /usr/local/docker/lazyliqiquan/data/files:/dev/shm/files \
#   go-judge

# ojserver
docker build -t owo-server -f Dockerfile.owo .
docker run -d --name owo --network oj_network \
    -p 5051:5051 --shm-size=256m --restart=always \
    -v /usr/local/docker/lazyliqiquan/data/files:/go/server/files \
    owo-server