# Этап 1. Сборка Go-бинарников
FROM golang:latest AS builder

WORKDIR /app

# Сначала модули (чтобы кешировать зависимости)
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем три бинарника: storage, analysis, gateway
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/storage ./cmd/storage
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/analysis ./cmd/analysis
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/gateway ./cmd/gateway

# Этап 2. Лёгкий рантайм
FROM alpine:3.20

WORKDIR /app

# Копируем собранные бинарники
COPY --from=builder /app/bin/* ./

# Копируем конфиг и init-sql (если вдруг понадобится)
COPY config ./config

# Пусть по умолчанию берём этот конфиг
ENV CONFIG_PATH=/app/config/local.yaml

# Откроем порты контейнера (документивно)
EXPOSE 8081 8069 8052

# По умолчанию запускать gateway (в docker-compose будем переопределять)
CMD ["./gateway"]
