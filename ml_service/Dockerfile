# ===== Stage 1: Builder =====
FROM python:3.11-slim AS builder

WORKDIR /app

# Установка системных зависимостей
RUN apt-get update && apt-get install -y \
    build-essential \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

COPY ml_service/requirements.txt .

RUN pip install --user --no-cache-dir -r requirements.txt

# Генерация gRPC кода
COPY proto/fm.proto ./proto/
RUN python -m grpc_tools.protoc \
    --proto_path=proto \
    --python_out=. \
    --grpc_python_out=. \
    proto/fm.proto


# ===== Stage 2: Runtime =====
FROM python:3.11-slim

WORKDIR /app

# Системные зависимости
RUN apt-get update && apt-get install -y \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /root/.local /root/.local
ENV PATH=/root/.local/bin:$PATH

COPY ml_service/*.py ./
COPY ml_service/entrypoint.sh ./
RUN chmod +x entrypoint.sh

# gRPC-файлы
COPY --from=builder /app/proto ./proto

EXPOSE 50051

CMD ["./entrypoint.sh"]