import os
from pathlib import Path
from concurrent import futures
import grpc
import time
import logging

import fm_pb2
import fm_pb2_grpc
from db import Database
from text_extractor import TextExtractor
from embedder import Embedder
from llm_client import LLMClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class QnAService(fm_pb2_grpc.QnAServicer):
    def __init__(self):
        dsn = os.environ.get("DATABASE_DSN", "")
        self.db = Database(dsn) if dsn else None
        self.extractor = TextExtractor()
        self.llm = LLMClient()
        if not self.db:
            logger.warning("DATABASE_DSN not set, running without DB")

    def UploadDocument(self, request, context):
        try:
            doc_id = f"doc_{int(time.time())}"
            
            if self.db:
                if request.file_bytes:
                    text = self.extractor.extract(request.file_bytes, request.filename)
                    logger.info(f"Extracted {len(text)} chars from {request.filename}")
                else:
                    text = request.text
                
                if not text:
                    logger.warning("No text extracted or provided")
                    return fm_pb2.UploadDocResponse(doc_id="", status="error: no text")
                
                self.db.save_document(
                    doc_id=doc_id,
                    user_id=request.user_id,
                    title=request.title,
                    filename=request.filename,
                )

                from embedder import Embedder
                embedder = Embedder()
                chunks = embedder.chunk_text(text, chunk_size=800, overlap=100)
                logger.info(f"Created {len(chunks)} chunks")

                embeddings = embedder.embed_batch(chunks)
                logger.info(f"Generated embeddings for {len(chunks)} chunks")

                chunk_data = []
                for i, (chunk_text, embedding) in enumerate(zip(chunks, embeddings)):
                    chunk_id = f"{doc_id}_chunk_{i}"
                    embedding_list = embedding.tolist()
                    chunk_data.append((chunk_id, doc_id, chunk_text, embedding_list))

                self.db.save_chunks(chunk_data)
                
            return fm_pb2.UploadDocResponse(doc_id=doc_id, status="ok")
        except Exception as e:
            logger.error(f"Upload failed: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return fm_pb2.UploadDocResponse(doc_id="", status="error")

    def Query(self, request, context):
        try:
            question = request.question.strip()
            if not question:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("Question is empty")
                return fm_pb2.QueryResponse(answer="", contexts=[])

            logger.info(f"Received query: {question}")

            contexts = []

            if self.db:
                embedder = Embedder()
                question_embedding = embedder.embed_text(question)
                results = self.db.search_chunks(request.user_id, question_embedding.tolist(), top_k=request.top_k or 5)
                logger.info(f"Search results: {len(results)} chunks")

                for chunk_id, chunk_text, score in results:
                    contexts.append(fm_pb2.Chunk(
                        chunk_id=chunk_id,
                        text=chunk_text,
                        score=float(score)
                    ))
            else:
                logger.warning("No database connected, using fallback context")
                fallback = fm_pb2.Chunk(
                    chunk_id="fallback_1",
                    text=f"Sample context for: {question}",
                    score=1.0
                )
                contexts = [fallback]

            context_texts = [c.text for c in contexts] if contexts else [f"No relevant data found for question: {question}"]

            logger.info(f"Calling LLM with {len(context_texts)} context(s)...")
            answer = self.llm.generate_answer(question, context_texts)
            logger.info(f"LLM returned {len(answer)} chars")

            return fm_pb2.QueryResponse(answer=answer, contexts=contexts)

        except Exception as e:
            logger.exception("Query failed")
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

def DirectQuery(self, request, context):
    """Answer without document context"""
    try:
        question = request.question.strip()
        logger.info(f"Direct query: {question}")
        
        # Пустой контекст = общий режим
        answer = self.llm.generate_answer(question, [])
        logger.info(f"Direct answer: {len(answer)} chars")
        
        return fm_pb2.QueryResponse(answer=answer, contexts=[])
    except Exception as e:
        logger.exception("Direct query failed")
        context.set_code(grpc.StatusCode.INTERNAL)
        return fm_pb2.QueryResponse(answer="", contexts=[])