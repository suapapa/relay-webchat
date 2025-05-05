#!/bin/bash
set -e
CR=icn.vultrcr.com/homincr1
IMAGE_TAG=$CR/webchat-relay:latest 
docker buildx build --platform linux/amd64 -f relay.dockerfile -t $IMAGE_TAG .
docker push $IMAGE_TAG
k rollout restart deployment webchat-relay