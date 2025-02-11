DB_URL ?= postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable
TEST_DB_URL=postgresql://test:test@localhost:5432/simple_bank?sslmode=disable
postgres:
	docker run --name postgres12 -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:12-alpine

createdb:
	docker exec postgres12 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec postgres12 dropdb simple_bank

create-migration:
	@read -p "Enter migration name: " name; \
	docker run --rm -v ./db/migration:/migration --network host migrate/migrate \
	 create -ext sql -dir /migrations -seq $${name}

migrateup:
	docker run --rm -v ./db/migration:/migration --network host migrate/migrate \
	  -path /migration -database $(DB_URL) -verbose up

migratedown:
	docker run --rm -v ./db/migration:/migration --network host migrate/migrate \
	  -path /migration -database $(DB_URL) -verbose down

sqlc:
	docker run --rm -v ./db:/db -v ./sqlc.yaml:/sqlc.yaml sqlc/sqlc generate

test-db-setup:
	docker cp ./db/test_db/setup.sql postgres12:/setup.sql
	docker exec -i postgres12 psql -U root -d simple_bank -f /setup.sql
	docker exec -i postgres12 rm /setup.sql
	make migrateup DB_URL="$(TEST_DB_URL)"

test-db-cleanup:
	docker cp ./db/test_db/cleanup.sql postgres12:/cleanup.sql
	docker exec -i postgres12 psql -U root -d simple_bank -f /cleanup.sql
	docker exec -i postgres12 rm /cleanup.sql

test: test-db-setup
	go test -v -cover ./...
	make test-db-cleanup

.PHONY: postgres createdb dropdb migrateup migratedown sqlc test
