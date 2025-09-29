.PHONY: check-env proto-gen up migrate

# Проверка .env файла (Windows-совместимая)
check-env:
	@if not exist .env ( \
		echo Error: .env file not found & \
		echo Please copy .env.example to .env and fill in your values & \
		echo Example: copy .env.example .env & \
		exit 1 \
	)
	@findstr "TELEGRAM_TOKEN" .env > nul || ( \
		echo Error: TELEGRAM_TOKEN not found in .env file & \
		echo Please edit .env file and add your actual Telegram token & \
		echo Get token from @BotFather & \
		exit 1 \
	)

# Генерация proto файлов
proto-gen:
	protoc --go_out=. --go-grpc_out=. -I proto proto/qna.proto
	python -m grpc_tools.protoc -I proto --python_out=ml_service --grpc_python_out=ml_service proto/qna.proto

# Запуск с проверкой .env файла
up: check-env
	docker-compose up --build

# Создание .env файла из примера
env-init:
	copy .env.example .env
	@echo Now edit .env file and add your Telegram token

# Запуск без проверки (если нужно)
up-force:
	docker-compose up --build

# Миграции БД
migrate:
	psql postgresql://app:pass@localhost:5432/appdb -f migrations/0001_init.sql

down:
	docker-compose down

logs:
	docker-compose logs -f