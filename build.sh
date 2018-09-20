#!/bin/bash

if [ $# != 1 ]; then
  echo "please input \"version\""
  echo "ex) $0 latest"
  exit 1
fi

./go_build_linux.sh

IMAGE=d2hub.com/d2hub-registryv2
VERSION=$1

docker build -t $IMAGE:$VERSION .
docker push $IMAGE:$VERSION
