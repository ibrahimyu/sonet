.PHONY: build run docker docker-compose clean test

# Build the application
build:
	go build -o bin/sonet ./cmd/api

# Run the application
run:
	go run ./cmd/api

# Build docker image
docker:
	docker build -t sonet .

# Run with docker-compose
docker-compose:
	docker-compose up -d

# Stop docker-compose
docker-compose-down:
	docker-compose down

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test -v ./...

# Create .env from .env.example if it doesn't exist
setup:
	test -f .env || cp .env.example .env
