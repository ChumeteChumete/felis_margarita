import grpc
import sys
import os

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(__file__))

import fm_pb2
import fm_pb2_grpc

def check_health():
    """Simple health check for ML service"""
    try:
        channel = grpc.insecure_channel('localhost:50051')
        stub = fm_pb2_grpc.QnAStub(channel)
        
        # Try a simple query
        request = fm_pb2.QueryRequest(
            user_id="health_check",
            question="test",
            top_k=1
        )
        
        response = stub.Query(request, timeout=5)
        print("✓ ML service is healthy")
        return 0
    except Exception as e:
        print(f"✗ ML service unhealthy: {e}")
        return 1

if __name__ == "__main__":
    sys.exit(check_health())