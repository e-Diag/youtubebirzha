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

# Проверяем, что сборка прошла успешно
RUN ls -la dist/ || (echo "ERROR: Frontend build failed!" && exit 1)
RUN test -f dist/index.html || (echo "ERROR: index.html not found in dist!" && exit 1)

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

# Проверяем, что index.html существует
RUN ls -la ./static/ || echo "Warning: static directory is empty"
RUN test -f ./static/index.html || (echo "ERROR: index.html not found after copy!" && exit 1)

# Билдим Go
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# 3. Финальный образ
FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add ca-certificates
COPY --from=backend-builder /app/backend/server .
COPY --from=backend-builder /app/backend/static ./static
# Копируем дополнительные статические файлы (terms.html, privacy.html) из корня проекта
COPY static/terms.html ./static/
COPY static/privacy.html ./static/
# Устанавливаем права доступа
RUN chmod -R 755 /app/static
# Финальная проверка
RUN ls -la /app/static/ && test -f /app/static/index.html || (echo "ERROR: index.html missing in final image!" && exit 1)
EXPOSE 8080
CMD ["./server"]
