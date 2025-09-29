import os
from concurrent import futures
import grpc
import time

import fm_pb2
import fm_pb2_grpc

class QnAService(fm_pb2_grpc.QnAServicer):
    def UploadDocument(self, request, context):
        doc_id = "doc_" + str(int(time.time()))
        return fm_pb2.UploadDocResponse(doc_id=doc_id, status="ok")

    def Query(self, request, context):
        #заглушка
        c = fm_pb2.Chunk(chunk_id="c1", text="Пример контекста для вопроса: "+request.question, score=0.9)
        return fm_pb2.QueryResponse(answer="", contexts=[c])

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=8))
    fm_pb2_grpc.add_QnAServicer_to_server(QnAService(), server)
    port = os.environ.get("GRPC_PORT", "50051")
    server.add_insecure_port("[::]:" + port)
    server.start()
    print("ML gRPC server started on", port)
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
