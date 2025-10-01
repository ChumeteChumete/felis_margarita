import os
os.environ.setdefault('DATABASE_DSN', 'postgresql://app:pass@localhost:5432/appdb')
from pathlib import Path
# from dotenv import load_dotenv

# env_path = Path(__file__).parent.parent / '.env'
# load_dotenv(dotenv_path=env_path)

from concurrent import futures
import grpc
import time
import logging

import fm_pb2
import fm_pb2_grpc
from db import Database

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class QnAService(fm_pb2_grpc.QnAServicer):
    def __init__(self):
        dsn = os.environ.get("DATABASE_DSN", "")
        self.db = Database(dsn) if dsn else None
        if not self.db:
            logger.warning("DATABASE_DSN not set, running without DB")

    def UploadDocument(self, request, context):
        try:
            doc_id = f"doc_{int(time.time())}"
            
            if self.db:
                # extract text from file_bytes or use plain text
                text = request.text if request.text else "extracted text placeholder"
                self.db.save_document(
                    doc_id=doc_id,
                    user_id=request.user_id,
                    title=request.title,
                    filename=request.filename,
                )
                # chunk and embed text will be implemented next
                
            return fm_pb2.UploadDocResponse(doc_id=doc_id, status="ok")
        except Exception as e:
            logger.error(f"Upload failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return fm_pb2.UploadDocResponse(doc_id="", status="error")

    def Query(self, request, context):
        try:
            # placeholder for now
            chunk = fm_pb2.Chunk(
                chunk_id="c1",
                text=f"Sample context for: {request.question}",
                score=0.9
            )
            return fm_pb2.QueryResponse(answer="", contexts=[chunk])
        except Exception as e:
            logger.error(f"Query failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return fm_pb2.QueryResponse(answer="", contexts=[])

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    fm_pb2_grpc.add_QnAServicer_to_server(QnAService(), server)
    
    port = os.environ.get("GRPC_PORT", "50051")
    server.add_insecure_port(f"[::]:{port}")
    server.start()
    
    logger.info(f"ML gRPC server running on port {port}")
    server.wait_for_termination()

if __name__ == "__main__":
    serve()