#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

mkdir -p logs/server logs/sql cookies checkpoints exports

export GF_GLOG_PATH="${GF_GLOG_PATH:-./logs/server}"
export GF_DATABASE_DEFAULT_LINK="${GF_DATABASE_DEFAULT_LINK:-admin:admin@tcp(127.0.0.1:3306)/go_crawler?charset=utf8mb4&parseTime=True&loc=Local}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

go run main.go
