from sentence_transformers import SentenceTransformer
import numpy as np
import logging

logger = logging.getLogger(__name__)

class Embedder:
    def __init__(self, model_name="sentence-transformers/all-MiniLM-L6-v2"):
        """
        Initialize embedding model.
        all-MiniLM-L6-v2 produces 384-dimensional embeddings
        """
        logger.info(f"Loading embedding model: {model_name}")
        self.model = SentenceTransformer(model_name)
        self.dimension = 384
        logger.info("Model loaded successfully")

    def embed_text(self, text):
        """
        Generate embedding for a single text
        Returns: numpy array of shape (384,)
        """
        embedding = self.model.encode(text, convert_to_numpy=True)
        return embedding

    def embed_batch(self, texts):
        """
        Generate embeddings for multiple texts
        Returns: numpy array of shape (n, 384)
        """
        embeddings = self.model.encode(texts, convert_to_numpy=True, show_progress_bar=False)
        return embeddings

    def chunk_text(self, text, chunk_size=500, overlap=50):
        """
        Split text into overlapping chunks for better context preservation
        
        Args:
            text: input text to chunk
            chunk_size: target size of each chunk in characters
            overlap: number of overlapping characters between chunks
        
        Returns: list of text chunks
        """
        if len(text) <= chunk_size:
            return [text]
        
        chunks = []
        start = 0
        
        while start < len(text):
            end = start + chunk_size
            

            if end < len(text):
                chunk_text = text[start:end]
                last_period = max(
                    chunk_text.rfind('. '),
                    chunk_text.rfind('? '),
                    chunk_text.rfind('! ')
                )
                if last_period > chunk_size - 100:
                    end = start + last_period + 1
            
            chunks.append(text[start:end].strip())
            start = max(end - overlap, start + 1)
        
        return chunks

    def embed_document(self, text):
        """
        Chunk document and generate embeddings for each chunk
        
        Returns: list of (chunk_text, embedding) tuples
        """
        chunks = self.chunk_text(text)
        embeddings = self.embed_batch(chunks)
        
        return list(zip(chunks, embeddings))