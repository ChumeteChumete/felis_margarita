import psycopg2
from psycopg2.extras import execute_values
import logging

logger = logging.getLogger(__name__)

class Database:
    def __init__(self, dsn):
        self.dsn = dsn
        self.conn = None
        self._connect()

    def _connect(self):
        try:
            self.conn = psycopg2.connect(self.dsn)
            logger.info("Connected to PostgreSQL")
        except Exception as e:
            logger.error(f"DB connection failed: {e}")
            raise

    def save_document(self, doc_id, user_id, title, filename):
        with self.conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO documents (id, user_id, title, filename)
                VALUES (%s, %s, %s, %s)
                ON CONFLICT (id) DO NOTHING
                """,
                (doc_id, user_id, title, filename)
            )
            self.conn.commit()

    def save_chunks(self, chunks):
        """
        chunks: list of (chunk_id, doc_id, text, embedding)
        """
        with self.conn.cursor() as cur:
            execute_values(
                cur,
                """
                INSERT INTO chunks (id, document_id, chunk_text, embedding)
                VALUES %s
                """,
                chunks
            )
            self.conn.commit()

    def search_chunks(self, user_id, embedding, top_k=5):
        """Search only user's own chunks"""
        with self.conn.cursor() as cur:
            cur.execute(
                """
                SELECT c.id, c.chunk_text, 1 - (c.embedding <=> %s::vector) as score
                FROM chunks c
                JOIN documents d ON c.document_id = d.id
                WHERE d.user_id = %s
                ORDER BY c.embedding <=> %s::vector
                LIMIT %s
                """,
                (embedding, user_id, embedding, top_k)
            )
            return cur.fetchall()