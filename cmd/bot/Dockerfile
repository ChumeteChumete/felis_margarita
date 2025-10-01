FROM golang:1.24-alpine AS builder

WORKDIR /build

# копируем го мод
COPY go.mod go.sum ./
RUN go mod download

# копируем источник
COPY . .

# билдим binory
RUN CGO_ENABLED=0 GOOS=linux go build -o bot ./cmd/bot

FROM alpine:3.18

WORKDIR /app

# устанавливаем ca-сертификаты для HTTPS
RUN apk --no-cache add ca-certificates

COPY --from=builder /build/bot .

ENTRYPOINT ["./bot"]
