#!/bin/bash

# 这个脚本使方便调试使用的，不应该上传到GitHub上

# docker chmod u+x restart.sh
docker stop owo
docker rm owo
docker rmi owo-server
# docker build -t owo-server -f Dockerfile.owo .
chmod u+x ./owoInit.sh
./owoInit.sh
docker ps