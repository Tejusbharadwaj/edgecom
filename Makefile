.PHONY: proto build test test-integration run docker-build docker-run docker-test clean

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/timeseries.proto

build: proto
	go mod tidy
	go build -o bin/time-series-service ./cmd/main.go

# Run unit tests only
test:
	go test ./... -tags=!integration

# Run integration tests only
test-integration:
	docker compose up --build db test

# Run locally without Docker
run: build
	./bin/time-series-service

# Docker commands
docker-build:
	docker compose build

docker-run:
	docker compose up --build app

docker-test:
	docker compose up --build db test

clean:
	rm -rf bin
	docker compose down --volumes