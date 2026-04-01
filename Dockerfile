# =====================================================
# 抖音电商罗盘爬虫系统 - Dockerfile
# =====================================================

# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装必要的依赖
RUN apk add --no-cache git make

# 设置 Go 代理（加速下载）
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译应用
RUN make build-linux

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/bin/linux_amd64/main /app/main

# 复制配置文件
COPY --from=builder /app/config.yaml /app/config.yaml

# 创建必要的目录
RUN mkdir -p logs cookies checkpoints exports

# 暴露端口
EXPOSE 8000

# 启动应用
CMD ["./main"]
