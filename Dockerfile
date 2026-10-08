FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend-builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
# .dockerignore excludes credentials and local tools but preserves tagged .gen build inputs.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates \
    && addgroup -g 10001 server \
    && adduser -D -u 10001 -G server server
COPY --from=backend-builder /app/server ./server
COPY --from=backend-builder --chown=10001:10001 /app/.gen/ ./.gen/
COPY --from=frontend-builder /app/frontend/build/ ./static/
# Startup must regenerate; a read-only root requires a writable mount at /app/.gen.
RUN chown -R 10001:10001 /app/.gen
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["./server"]
