# 构建阶段
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o UniBarrage .

# 运行阶段
FROM alpine:latest

# 安装 ca-certificates（用于 HTTPS 请求）和 bash（用于启动脚本）
RUN apk --no-cache add ca-certificates bash

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/UniBarrage .

# 创建启动脚本，支持 AUTH_TOKEN 环境变量
RUN echo '#!/bin/bash' > /app/start.sh && \
    echo 'ARGS="-apiHost 0.0.0.0 -wsHost 0.0.0.0"' >> /app/start.sh && \
    echo '[ -n "$AUTH_TOKEN" ] && ARGS="$ARGS -authToken $AUTH_TOKEN"' >> /app/start.sh && \
    echo 'exec ./UniBarrage $ARGS "$@"' >> /app/start.sh && \
    chmod +x /app/start.sh

# 暴露端口
EXPOSE 8080 7777 8888

# 使用启动脚本
ENTRYPOINT ["/app/start.sh"]
