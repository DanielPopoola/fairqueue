FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go run github.com/swaggo/swag/cmd/swag@latest init -g internal/api/server.go -o docs/api/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /fairqueue ./cmd/api


FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /fairqueue ./fairqueue

EXPOSE 8080

ENTRYPOINT ["./fairqueue"]