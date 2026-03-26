# ---------- BUILD STAGE ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git (needed for private modules or proto module)
RUN apk add --no-cache git

# Copy go.mod and go.sum first for layer caching
COPY services/sfu/go.mod services/sfu/go.sum ./
RUN go mod download

# Copy entire SFU source
COPY services/sfu ./

# Build the SFU binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -o app ./main

# ---------- RUNTIME STAGE ----------
FROM alpine:3.20

WORKDIR /app

# Runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy complied binary from builder
COPY --from=builder /app/app .

# Expose gRPC port
EXPOSE 50051

CMD ["./app"]
