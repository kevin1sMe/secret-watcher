# 使用官方的golang镜像作为构建环境
FROM golang:1.20-alpine AS builder

# 设置工作目录
WORKDIR /app

# 将当前目录的所有文件复制到工作目录中
COPY . .

# 获取依赖
RUN go mod tidy

# 构建可执行文件
RUN go build -o secret-watcher main.go

# ----------------------------

# 使用轻量级的alpine镜像作为运行环境
FROM alpine:latest

# 安装必要的依赖
RUN apk --no-cache add ca-certificates

# 设置工作目录
WORKDIR /root/

# 从构建环境复制可执行文件到当前目录
COPY --from=builder /app/secret-watcher .

# 运行可执行文件
CMD ["./secret-watcher", "--config", "/config/config.yaml"]
