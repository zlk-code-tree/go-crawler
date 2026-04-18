#!/bin/bash
APP_NAME="myapp"
VERSION="v1.0.1"

echo "🚀 Building for Linux (amd64)..."

# 核心编译命令
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -o ${APP_NAME}-${VERSION}-linux-amd64 .

echo "✅ Build success! File: ${APP_NAME}-${VERSION}-linux-amd64"