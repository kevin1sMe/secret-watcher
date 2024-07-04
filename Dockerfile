# 使用官方的golang镜像作为构建环境
FROM golang:1.20-alpine AS builder

RUN apk --update add ca-certificates upx


ENV CGO_ENABLED=0

# 设置工作目录
WORKDIR /app

# 将当前目录的所有文件复制到工作目录中
COPY . .

# 获取依赖
RUN go mod tidy

# 构建可执行文件
RUN go build -ldflags "-w -s" -o secret-watcher main.go
RUN upx -9 -o secret-watcher.minify secret-watcher && \
    chmod +x secret-watcher.minify

# ----------------------------

# 使用轻量级的alpine镜像作为运行环境
FROM alpine:latest
LABEL maintainer "kevinlin <linjiang1205@qq.com>"

# 设置工作目录
WORKDIR /root/

# 从构建环境复制可执行文件到当前目录
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/secret-watcher.minify secret-watcher

# 运行可执行文件
CMD ["./secret-watcher", "--config", "/config/config.yaml"]
