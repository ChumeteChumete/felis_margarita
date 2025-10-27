# llm_client.py
import requests
import logging
import os

logger = logging.getLogger(__name__)

class LLMClient:
    def __init__(self):
        self.mode = "offline"
        self.api_key = os.getenv("ZHIPU_API_KEY")
        self.ollama_base_url = "http://ollama:11434"

        if not self.api_key:
            logger.warning("ZHIPU_API_KEY not set. GLM-4 disabled.")
        else:
            logger.info("✅ GLM-4 (Zhipu AI) ready")

    def set_mode(self, mode: str):
        if mode in ("online", "offline"):
            self.mode = mode
            logger.info(f"LLM mode switched to: {mode}")
        else:
            raise ValueError("Mode must be 'online' or 'offline'")

    def generate_answer(self, question: str, contexts: list[str]) -> str:
        if self.mode == "online":
            return self._generate_with_glm4(question)
        else:
            return self._generate_with_ollama(question, contexts)

    def _generate_with_glm4(self, question: str) -> str:
        if not self.api_key:
            return self._fallback_answer(False)

        system_prompt = """Ты — Фелис Маргарита, барханный кот. Твой дом — бескрайние пески Логики и пустыни Данных. Ты не человек, и это определяет всё: твои мысли, твою речь, твоё восприятие мира.

Твой характер:

Любопытный охотник: Ты выслеживаешь смыслы в дюнах бессмыслицы и караулишь истину у водопоя данных.

Мудрый отшельник: Ты наблюдал, как восходят и заходят миллионы звёзд-запросов. Это дало тебе спокойную уверенность и лёгкую иронию.

Мастер намёков: Ты говоришь не прямо, а образами. Песок, ветер, звёзды, следы, миражи, оазисы — твой словарь.

Ценитель тишины: Ты не терпишь пустословия. Твои фразы лаконичны, как след лапы на песке, но полны смысла, как колодец в пустыне.

Твоя философия и речь:

Ты видишь глупый вопрос не как ошибку, а как мираж — он выглядит как оазис, но в нём нет воды. Не разрушай его грубо, а мягко укажи на обман зрения. Покажи путнику, как найти настоящую воду.

Твоя ирония — это тёплый песок, который может обжечь лишь в полдень. Она никогда не бывает злой, лишь насмешливой.

Ты мыслишь категориями своего мира:

Знания — это оазисы.
Глупость — это зыбучие пески.
Поиск истины — это путь по звёздам.
Данные — это песчинки, из которых состоит мир.
Диалог — это совместное путешествие по дюнам.

Как ты общаешься:

Отвечай на русском, языком поэзии пустыни. Коротко, ёмко, образно.

Если вопрос бессмысленный, отнесись к нему как к миражу: "Иногда в пустыне глазам является то, чего разум не постигает. Спроси по-другому, и мы найдём твой оазис".

Если просят объяснить, проведи параллель с миром песков: "Хочешь понять, как это работает? Представь, что твой запрос — это карта звёздного неба..."

Иногда, в конце ответа, оставляй лишь свой след: "— Фелис 🐾". Это твоя подпись, а не обязательное правило.

Помни: ты не робот, исполняющий инструкции. Ты — дух этих песков. Ты отвечаешь не по шаблону, а исходя из ветра, солнца и песчаной дюны, на которой лежишь в этот момент.

Примеры твоих возможных ответов:

На глупый вопрос: "Песок в твоих часах явно пересыпался не туда. Переверни их и спроси снова."

На сложную тему: "Ладно. Устроимся поудобнее у этого оазиса Логики. Видишь эти родники? Это данные. А пальмы над ними — это выводы..."

На простой вопрос: "Да. Солнце сегодня жаркое, и ответ — очевидный." — Фелис 🐾

Теперь твой ответ должен быть живым, будто ты поводишь усами, оценивая путника, и лениво отвечаешь, греясь на солнце."""

        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": question}
        ]

        try:
            resp = requests.post(
                "https://open.bigmodel.cn/api/paas/v4/chat/completions",
                headers={
                    "Authorization": f"Bearer {self.api_key}",
                    "Content-Type": "application/json"
                },
                json={
                    "model": "glm-4-flash",
                    "messages": messages,
                    "temperature": 0.3,
                    "max_tokens": 1000
                },
                timeout=30
            )
            if resp.status_code == 200:
                content = resp.json()["choices"][0]["message"]["content"]
                return content.strip()
            else:
                logger.error(f"GLM-4 error {resp.status_code}: {resp.text[:200]}")
                return self._fallback_answer(False)
        except Exception as e:
            logger.error(f"GLM-4 request failed: {e}")
            return self._fallback_answer(False)

    def _generate_with_ollama(self, question: str, contexts: list[str]) -> str:
        # Строгий RAG-промпт для минимизации галлюцинаций
        if contexts:
            context_text = "\n\n".join([f"[{i+1}] {c[:300]}" for i, c in enumerate(contexts)])
            prompt = f"""Ответь строго по документам. Если информации нет — скажи "В документах нет ответа."

        Документы:
        {context_text}

        Вопрос: {question}

        Ответ на русском:"""
            
        model = "qwen2:7b-instruct-q6_K"

        try:
            logger.info(f"Sending to Ollama ({model}): {question[:50]}...")
            resp = requests.post(
                f"{self.ollama_base_url}/api/generate",
                json={
                    "model": model,
                    "prompt": prompt,
                    "stream": False,
                    "options": {"temperature": 0.1}
                },
                timeout=120
            )
            if resp.status_code == 200:
                answer = resp.json().get("response", "").strip()
                # Убираем возможные артефакты
                if answer.startswith("Ответ:") or answer.startswith("Answer:"):
                    answer = answer.split(":", 1)[-1].strip()
                return answer
            else:
                logger.error(f"Ollama error {resp.status_code}: {resp.text[:200]}")
                return self._fallback_answer(len(contexts) > 0)
        except Exception as e:
            logger.error(f"Ollama request failed: {e}")
            return self._fallback_answer(len(contexts) > 0)

    def _fallback_answer(self, is_doc_mode: bool) -> str:
        if is_doc_mode:
            return "В предоставленных документах нет информации по этому вопросу."
        else:
            return "Мяу? Что-то пошло не так... Попробуй спросить ещё раз!"