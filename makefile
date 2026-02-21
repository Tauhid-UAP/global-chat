include .env
export

.PHONY: up down build rebuild logs migrate-up migrate-down restart web infra

# Start everything (DB + Redis + Web)
up:
	docker compose up -d

# Start only infrastructure
infra:
	docker compose up -d db redis

# Start web only
web:
	docker compose up web

# Stop everything
down:
	docker compose down

# Build images
build:
	docker compose build

# Force rebuild
rebuild:
	docker compose build --no-cache

# View logs
logs:
	docker compose logs -f

# Run migrations up
migrate-up:
	docker compose --profile tools run --rm migrate \
		-path=/migrations -database=$$DATABASE_URL up

# Rollback one migration
migrate-down:
	docker compose --profile tools run --rm migrate \
		-path=/migrations -database=$$DATABASE_URL down 1

# Restart web container
restart:
	docker compose restart web
