.PHONY: proto up up-d down logs logs-bot logs-ml migrate env-init clean rebuild test

# Generate proto files
proto:
	@if not exist "pkg\proto" mkdir pkg\proto
	protoc --go_out=./pkg/proto --go-grpc_out=./pkg/proto --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative -I proto proto/fm.proto
	python -m grpc_tools.protoc -I proto --python_out=ml_service --grpc_python_out=ml_service proto/fm.proto

# Start all services
up:
	@if not exist .env (echo Error: .env file not found. Run 'make env-init' first && exit /b 1)
	docker compose up --build

# Start in detached mode
up-d:
	@if not exist .env (echo Error: .env file not found. Run 'make env-init' first && exit /b 1)
	docker compose up --build -d

# Stop all services
down:
	docker compose down

# Stop and remove volumes
clean:
	docker compose down -v

# Rebuild without cache
rebuild:
	docker compose build --no-cache
	docker compose up

# View all logs
logs:
	docker compose logs -f

# View bot logs
logs-bot:
	docker compose logs -f bot

# View ML service logs
logs-ml:
	docker compose logs -f ml-service

# Run database migrations manually
migrate:
	docker compose exec postgres psql -U app -d appdb -f /migrations/0001_init.sql

# Run tests
test:
	go test -v ./...

# Create .env from example
env-init:
	@if exist .env (echo .env already exists) else (copy .env.example .env && echo Edit .env and add your TELEGRAM_TOKEN)