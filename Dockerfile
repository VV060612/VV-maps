# 阶段 1: 构建阶段
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置 Go 代理，加快下载速度
ENV GOPROXY=https://goproxy.cn,direct

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用，生成名为 main 的二进制文件
RUN go build -o main .

# 阶段 2: 运行阶段 (使用极小的镜像)
FROM alpine:latest

# 安装基础库 (gin 有时需要 ca-certificates)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 复制静态资源和地图数据 (非常重要！)
COPY --from=builder /app/static ./static
COPY --from=builder /app/map_data.json .

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./main"]