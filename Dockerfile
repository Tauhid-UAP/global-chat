# ---------- BUILD STAGE ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o app ./cmd/web


# ---------- RUNTIME STAGE ----------
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Install golang-migrate CLI (pinned version)
ENV MIGRATE_VERSION=v4.17.0

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz \
    -o migrate.tar.gz \
    && tar -xzf migrate.tar.gz \
    && mv migrate /usr/local/bin/migrate \
    && chmod +x /usr/local/bin/migrate \
    && rm migrate.tar.gz

# Copy application binary
COPY --from=builder /app/app .

# Copy migrations into image
COPY --from=builder /app/migrations ./migrations

EXPOSE 8000

CMD ["./app"]
