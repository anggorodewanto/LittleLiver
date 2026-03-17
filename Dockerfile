# Stage 1: Build frontend
FROM node:20-slim AS frontend
WORKDIR /app/frontend
COPY frontend/ .
RUN npm ci && npm run build

# Stage 2: Build backend
FROM golang:1.26 AS backend
WORKDIR /app
COPY backend/ ./backend/
WORKDIR /app/backend
RUN CGO_ENABLED=1 go build -o /server ./cmd/server

# Stage 3: Runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates imagemagick && rm -rf /var/lib/apt/lists/*
COPY --from=backend /server /server
COPY --from=frontend /app/frontend/build /static
ENV STATIC_DIR=/static
EXPOSE 8080
CMD ["/server"]
