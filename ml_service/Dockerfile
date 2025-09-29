FROM python:3.11-slim

WORKDIR /app

# Копируем proto файлы
COPY proto ./proto

# Копируем requirements и устанавливаем зависимости
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Генерируем Python gRPC файлы из proto
RUN python -m grpc_tools.protoc \
    --proto_path=proto \
    --python_out=. \
    --grpc_python_out=. \
    proto/fm.proto

# Копируем остальные файлы проекта
COPY . .

# Создаем __init__.py для корректной работы импортов
RUN touch __init__.py

EXPOSE 50051

CMD ["python", "server.py"]