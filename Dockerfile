FROM golang:1.27-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server main.go

FROM alpine:3.24
WORKDIR /app

RUN apk --no-cache add tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime && \
    echo "Asia/Tokyo" > /etc/timezone && \
    adduser -D -H -u 10001 appuser

COPY --from=builder /app/server .

ENV PORT=8080

USER appuser

CMD ["./server"]