# ===== Stage 1: Builder =====
FROM python:3.11-slim AS builder

WORKDIR /app

# Ставим только то, что нужно для сборки зависимостей
RUN apt-get update && apt-get install -y \
    build-essential \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Копируем requirements отдельно
COPY ml_service/requirements.txt .
RUN pip install --user --no-cache-dir -r requirements.txt

# Копируем proto и генерируем gRPC код
COPY proto/fm.proto ./proto/
RUN python -m grpc_tools.protoc \
    --proto_path=proto \
    --python_out=. \
    --grpc_python_out=. \
    proto/fm.proto


# ===== Stage 2: Runtime =====
FROM python:3.11-slim

WORKDIR /app

# Копируем зависимости из builder
COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH

# Устанавливаем только postgres client (для healthcheck и миграций, если нужен)
RUN apt-get update && apt-get install -y \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Копируем код приложения
COPY ml_service/*.py ./
COPY ml_service/entrypoint.sh ./
RUN chmod +x entrypoint.sh

# Копируем сгенерированные grpc файлы из builder
COPY --from=builder /app/proto ./proto

EXPOSE 50051

CMD ["./entrypoint.sh"]
