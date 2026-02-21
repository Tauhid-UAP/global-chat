include .env

migrate-up:
	docker compose run --rm migrate \
		-path /migrations \
		-database "$(DATABASE_URL)" up

migrate-down:
	docker compose run --rm migrate \
		-path /migrations \
		-database "$(DATABASE_URL)" down 1


