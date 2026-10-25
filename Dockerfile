FROM golang:1.27-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0: OS のライブラリに依存しない静的バイナリにする
# -tags timetzdata: タイムゾーン情報をバイナリに埋め込む
RUN CGO_ENABLED=0 go build -tags timetzdata -trimpath -o server main.go

# 実行イメージ：何も入っていない scratch にバイナリだけを置く
FROM scratch
WORKDIR /app

COPY --from=builder /app/server .

ENV TZ=Asia/Tokyo
ENV PORT=8080

# scratch にはユーザー情報がないため、数値の UID/GID で一般ユーザーとして実行する
USER 10001:10001

CMD ["./server"]