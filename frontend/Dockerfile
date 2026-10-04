# syntax=docker/dockerfile:1
# Build context = root repo (xem deployments/docker/docker-compose.yml)

# ---------------------------------------------------------------- builder
FROM node:22-alpine AS builder
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./

# Cấu hình soketi nhúng vào bundle lúc build (Vite đọc process.env.VITE_*).
# App key là công khai; app secret chỉ nằm ở identity/notification.
ARG VITE_SOKETI_APP_KEY=task-manager
ARG VITE_SOKETI_WS_PATH=/api/soketi
ARG VITE_SOKETI_CLUSTER=mt1
ARG VITE_NOTIFICATION_API_URL=/api/notification
ENV VITE_SOKETI_APP_KEY=$VITE_SOKETI_APP_KEY \
    VITE_SOKETI_WS_PATH=$VITE_SOKETI_WS_PATH \
    VITE_SOKETI_CLUSTER=$VITE_SOKETI_CLUSTER \
    VITE_NOTIFICATION_API_URL=$VITE_NOTIFICATION_API_URL

RUN npm run build

# ---------------------------------------------------------------- runtime
FROM nginx:1.27-alpine

COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80 3000
