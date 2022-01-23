#!/bin/sh

set -e

docker login -u foochu

go test ./...

docker build -t bgweb-api .

GIT_HASH=$(git rev-parse --short HEAD)

docker tag bgweb-api foochu/bgweb-api:${GIT_HASH}
docker tag bgweb-api foochu/bgweb-api:latest

docker push foochu/bgweb-api:${GIT_HASH}
docker push foochu/bgweb-api:latest
