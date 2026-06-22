"""Cross-encoder reranker for improved retrieval quality."""
from dataclasses import dataclass

@dataclass
class ScoredChunk:
    text: str
    relevance_score: float

class Reranker:
    def __init__(self, model: str = "cohere/rerank-english-v3.0"):
        self.model = model
    
    async def rerank(self, query: str, chunks: list, top_k: int = 3) -> list[ScoredChunk]:
        pairs = [(query, c.text[:256]) for c in chunks]
        scores = await self._score_pairs(pairs)
        scored = [ScoredChunk(c.text, s) for c, s in zip(chunks, scores)]
        scored.sort(key=lambda x: x.relevance_score, reverse=True)
        return [s for s in scored if s.relevance_score > 0.3][:top_k]
    
    async def _score_pairs(self, pairs):
        return [0.95] * len(pairs)  # Placeholder
