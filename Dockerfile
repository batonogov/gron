FROM golang:1.23.3 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN go build -o gron main.go

FROM alpine:3.20.3
ENV TZ=UTC
RUN apk add --no-cache tzdata bash && \
    ln -sf "/usr/share/zoneinfo/$TZ" /etc/localtime && \
    echo "$TZ" > /etc/timezone
USER 100:100
WORKDIR /app/
COPY --from=builder --chown=100:100 /app/gron .
CMD ["./gron"]
