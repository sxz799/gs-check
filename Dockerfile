FROM node:16

WORKDIR /gsCheck-web/
COPY ./web/ .

RUN npm install && npm run build


# 使用官方 Golang 镜像作为基础镜像
FROM golang:1.21-alpine as builder

# 设置工作目录
WORKDIR /go/src/github.com/sxz799/gsCheck-server

RUN apk --no-cache add gcc musl-dev

# 将应用的代码复制到容器中
COPY ./server/ .
COPY --from=0 /gsCheck-web/dist/ ./dist

# 编译应用程序
RUN go env -w GO111MODULE=on \
    && go mod tidy \
    && go build -o app .

FROM alpine:latest

WORKDIR /home

RUN apk --no-cache add bash

RUN apk update && apk add tzdata 
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime 
RUN echo "Asia/Shanghai" > /etc/timezone

COPY --from=1 /go/src/github.com/sxz799/gsCheck-server/app ./
COPY --from=1 /go/src/github.com/sxz799/gsCheck-server/conf.yaml ./

EXPOSE 7990

# 运行应用程序
CMD ["./app"]