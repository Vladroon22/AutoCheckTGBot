FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ./bot cmd/bot/main.go 

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/.env .env
COPY --from=builder /app/bot ./bot

CMD ["./bot"]