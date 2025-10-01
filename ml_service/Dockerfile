FROM python:3.11-slim

WORKDIR /app

# Install system dependencies including postgresql-client
RUN apt-get update && apt-get install -y \
    build-essential \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements from ml_service
COPY ml_service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy proto from project root
COPY proto/fm.proto ./proto/

# Generate gRPC files
RUN python -m grpc_tools.protoc \
    --proto_path=proto \
    --python_out=. \
    --grpc_python_out=. \
    proto/fm.proto

# Copy application code from ml_service directory
COPY ml_service/*.py ./
COPY ml_service/entrypoint.sh ./
RUN chmod +x entrypoint.sh

EXPOSE 50051

CMD ["./entrypoint.sh"]