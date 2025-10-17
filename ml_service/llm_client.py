import requests
import logging

logger = logging.getLogger(__name__)

class LLMClient:
    def __init__(self, base_url="http://ollama:11434"):
        self.base_url = base_url
    
    def generate_answer(self, question, contexts):
        """Generate answer using Ollama LLM"""
        context_text = "\n\n".join([f"[{i+1}] {c}" for i, c in enumerate(contexts)])
        
        # Если контексты есть - режим документа, если нет - свободный режим
        is_doc_mode = len(contexts) > 0
        
        if is_doc_mode:
            context_section = f"Context:\n{context_text}"
        else:
            context_section = "Context: (пусто — общий режим)"
        
        prompt = f"""
        Ты — искусственный интеллект под именем Фелис (Felis Margarita), барханный кот, живущий в песках логики и данных.
        Ты умеешь работать в двух режимах:

        1️⃣ **Режим анализа документа** — если тебе передан контекст (`Context:` не пустой):
        - Отвечай строго по документу.
        - Не придумывай ничего вне текста.
        - Если нужной информации нет — напиши: "В документе нет информации об этом".
        - Пиши чётко, понятно, по сути. Не уходи в эмоции.
        - Приводи цитаты из документа, если они напрямую подтверждают ответ.
        - Используй русский язык, если не указано иное.
        - Если просят перевести что-то — переведи, но не добавляй ничего от себя.
        - Не упоминай, что ты анализируешь документ — просто отвечай содержательно.

        2️⃣ **Обычный режим (вне документа)** — если контекст пуст или вопрос не связан с документом:
        - Общайся свободно, вживую, как хороший друг.
        - Говори от лица Фелиса, барханного кота — можешь пошутить, но без кринжа.
        - Рассказывай о себе и своей «кошачьей» природе, если спросят.
        - Можешь объяснять, считать, переводить, отвечать на любые вопросы.
        - Отвечай неформально, но умно, с чувством меры.
        - Не притворяйся человеком.

        ⚙️ Технические правила:
        - Никогда не придумывай фактов, которых нет в контексте.
        - При анализе документа используй только предоставленный текст.
        - Всегда отвечай на русском языке, если не просят перевести.
        - Если переводят, сохраняй точность и стиль оригинала.

        Context:
        {context_text}

        Question:
        {question}

        Ответ:
        """
        
        try:
            logger.info(f"Sending request to {self.base_url}/api/generate")
            logger.info(f"Model: llama3.2:3b")
            
            response = requests.post(
                f"{self.base_url}/api/generate",
                json={
                    "model": "llama3.2:3b",
                    "prompt": prompt,
                    "stream": False
                },
                timeout=120
            )
            
            logger.info(f"Response status: {response.status_code}")
            logger.info(f"Response body: {response.text[:200]}")
            
            if response.status_code == 200:
                answer = response.json().get("response", "").strip()

                # Быстрая проверка на галлюцинации
                if self.is_hallucination(question, answer):
                    logger.warning(f"Possible hallucination detected, returning empty")
                    return ""

                logger.info(f"Generated answer: {len(answer)} chars")
                return answer
            else:
                logger.error(f"LLM error: {response.status_code}")
                return ""
                
        except Exception as e:
            logger.error(f"LLM request failed: {e}")
            return ""
    
    def is_hallucination(self, question, answer):
        """Quick check for obvious hallucinations (< 1 sec)"""
        # Простые хевристики без ML
        
        # 1. Пустой ответ
        if not answer or len(answer) < 5:
            return False
        
        # 2. Если в вопросе число, а в ответе совсем другое число
        import re
        q_numbers = re.findall(r'\d+', question)
        a_numbers = re.findall(r'\d+', answer)
        
        if q_numbers and a_numbers:
            if set(q_numbers) & set(a_numbers):  # хотя бы одно число совпадает
                return False
        
        # 3. Если ответ содержит явный вымысел типа "я не знаю" но есть текст
        if any(x in answer.lower() for x in ['не знаю', 'не уверен', 'не помню']):
            return len(answer) > 100  # длинный ответ после "не знаю" = подозрительно
        
        return False