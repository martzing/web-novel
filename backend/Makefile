.PHONY: run tidy build migrate seed db-up

tidy:
	go mod tidy

build:
	go build -o bin/api     ./cmd/api
	go build -o bin/migrate ./cmd/migrate

run: build
	./bin/api

migrate: build
	./bin/migrate -cmd up

migrate-down: build
	./bin/migrate -cmd down

status: build
	./bin/migrate -cmd status
