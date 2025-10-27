import redis
import pickle
import hashlib
import logging

logger = logging.getLogger(__name__)

class RedisCache:
    def __init__(self, host="redis", port=6379, db=0):
        try:
            self.client = redis.Redis(host=host, port=port, db=db, decode_responses=False)
            self.client.ping()
            logger.info("✅ Redis connected")
        except Exception as e:
            logger.warning(f"Redis unavailable: {e}")
            self.client = None
    
    def _make_key(self, text):
        return f"emb:{hashlib.md5(text.encode()).hexdigest()}"
    
    def get_embedding(self, text):
        if not self.client:
            return None
        try:
            key = self._make_key(text)
            data = self.client.get(key)
            if data:
                return pickle.loads(data)
        except Exception as e:
            logger.error(f"Redis get failed: {e}")
        return None
    
    def set_embedding(self, text, embedding, ttl=86400):  # 24 hours
        if not self.client:
            return
        try:
            key = self._make_key(text)
            self.client.setex(key, ttl, pickle.dumps(embedding))
        except Exception as e:
            logger.error(f"Redis set failed: {e}")