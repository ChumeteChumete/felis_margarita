FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY . .
RUN apk add --no-cache git
RUN go build -o /app ./cmd/bot

FROM alpine:3.18
COPY --from=builder /app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
