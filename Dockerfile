FROM golang:alpine AS builder

WORKDIR /build

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --update --no-cache coreutils && \
    apk add --no-cache curl && \
    apk add --no-cache make git


ENV GOPROXY https://goproxy.cn
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux make build .

FROM alpine:3
ENV LANG=C.UTF-8
ENV TZ=Asia/Shanghai

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --update --no-cache coreutils && \
    apk add --no-cache curl 

WORKDIR /app
COPY --from=builder /build/dist/sync-image /app/

ENTRYPOINT ["./sync-image"]