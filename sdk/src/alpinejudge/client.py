import json
import httpx
from typing import AsyncGenerator, Optional, Literal

from .models import JudgeEvent

LanguageType = Literal["c", "cpp", "java", "python", "go", "js"]

class AlpineJudge:

    def __init__(self, base_urlstr = "http://localhost:1111", timeout: float = 300.0) -> None:
        self.base_url = base_urlstr.rstrip("/")
        self.timeout = httpx.Timeout(timeout=timeout, connect=10.0)
        self._client: Optional[httpx.AsyncClient] = None

    async def __aenter__(self):
        """Context manager support: async with AsyncAlpineJudge() as client:"""
        self._client = httpx.AsyncClient(base_url=self.base_url, timeout=self.timeout)
        return self

    async def __aexit__(self, exc_type, exc, tb):
        if self._client:
            await self._client.aclose()


    @property
    def client(self) -> httpx.AsyncClient:
        """Fallback if context manager isn't used explicitly."""
        if self._client is None or self._client.is_closed:
            self._client = httpx.AsyncClient(base_url=self.base_url, timeout=self.timeout)
        return self._client

    async def close(self):
        """Explicitly close the client session."""
        if self._client and not self._client.is_closed:
            await self._client.aclose()

    async def submit(
        self,
        submission_id: str,
        bucket: str,
        language: LanguageType,
        source: str,
        testset_id: str,
        memory_limit_mb: int = 512,
        timeout_sec: int = 60,
        log_limit_kb: int = 256,
    ) -> dict:
        """Sends submission asynchronously."""
        payload = {
            "submission_id": submission_id,
            "bucket": bucket,
            "language": language,
            "source": source,
            "testset_id": testset_id,
            "memory_limit_mb": memory_limit_mb,
            "timeout_sec": timeout_sec,
            "log_limit_kb": log_limit_kb,
        }
        res = await self.client.post("/submit", json=payload)
        res.raise_for_status()
        return res.json()

    async def listen_events(self, submission_id: str) -> AsyncGenerator[JudgeEvent, None]:
        """
        Async Generator yielding live JudgeEvents over SSE.
        Terminates automatically on Type == "RESULT".
        """
        url = f"/submissions/{submission_id}/events"
        headers = {"Accept": "text/event-stream"}

        # Open non-blocking streaming GET request
        async with self.client.stream("GET", url, headers=headers) as response:
            response.raise_for_status()

            # Iterate asynchronously over text lines
            async for line in response.aiter_lines():
                if not line:
                    continue

                line_str = line.strip()

                # Parse standard SSE payload lines
                if line_str.startswith("data:"):
                    raw_data = line_str[5:].strip()
                    if not raw_data:
                        continue

                    event_dict = json.loads(raw_data)
                    event = JudgeEvent.from_dict(event_dict)

                    yield event

                    # Stop listening when backend signals final result
                    if event.type == "RESULT":
                        break


    async def submit_and_watch(
        self,
        submission_id: str,
        bucket: str,
        language: LanguageType,
        source: str,
        testset_id: str,
        memory_limit_mb: int = 512,
        timeout_sec: int = 60,
        log_limit_kb: int = 256,
    ) -> AsyncGenerator[JudgeEvent, None]:
        """Convenience method: Submits and immediately streams results."""
        await self.submit(
            submission_id, bucket, language, source, testset_id, memory_limit_mb, timeout_sec, log_limit_kb
        )
        async for event in self.listen_events(submission_id):
            yield event