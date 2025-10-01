# From project root:
python -m pip install grpcio-tools
python -m grpc_tools.protoc -I./proto --python_out=./ml_service --grpc_python_out=./ml_service proto/fm.proto
