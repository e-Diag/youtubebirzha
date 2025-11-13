# 1. Фронтенд
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

# Копируем package.json
COPY frontend/package*.json ./

# Устанавливаем зависимости
RUN npm ci

# Копируем исходники
COPY frontend/ ./

# Даём права на выполнение
RUN chmod +x node_modules/.bin/vite

# Билдим
RUN npm run build

# 2. Бэкенд
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app/backend

# Устанавливаем git
RUN apk add --no-cache git

# Копируем go.mod
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Копируем код
COPY backend/ ./

# Копируем статику из фронтенда
COPY --from=frontend-builder /app/frontend/dist ./static

# Билдим Go
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# 3. Финальный образ
FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates
COPY --from=backend-builder /app/backend/server .
COPY --from=backend-builder /app/backend/static ./static
EXPOSE 8080
CMD ["./server"]