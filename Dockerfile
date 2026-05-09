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
RUN apt-get update && apt-get install -y ca-certificates imagemagick ghostscript && rm -rf /var/lib/apt/lists/*
# Debian's default ImageMagick policy blocks PDF reads (CVE-2018-16509 hardening).
# We rasterize PDFs via Ghostscript to generate first-page thumbnails for imaging
# studies — that flow is trusted (server-controlled input that already passed
# upload-size/MIME validation), so re-enable PDF/PS read+write.
RUN if [ -f /etc/ImageMagick-6/policy.xml ]; then \
      sed -i 's#rights="none" pattern="PDF"#rights="read|write" pattern="PDF"#' /etc/ImageMagick-6/policy.xml; \
      sed -i 's#rights="none" pattern="PS"#rights="read|write" pattern="PS"#' /etc/ImageMagick-6/policy.xml; \
    fi
COPY --from=backend /server /server
COPY --from=backend /app/backend/migrations /migrations
COPY --from=frontend /app/frontend/build /static
ENV STATIC_DIR=/static
ENV MIGRATIONS_DIR=/migrations
EXPOSE 8080
CMD ["/server"]
