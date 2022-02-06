#!/bin/sh

set -e

cd ./cmd/bgweb-api
swag init --parseDependency --parseDepth 1
