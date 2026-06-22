"""Idempotent event consumer for at-least-once delivery."""
import json
from datetime import datetime

class IdempotentConsumer:
    def __init__(self, redis_client, ttl: int = 86400):
        self.processed = redis_client
        self.ttl = ttl
    
    async def process(self, event: dict):
        event_id = event.get("id")
        if not event_id:
            raise ValueError("Event missing id")
        
        if await self.processed.exists(event_id):
            print(f"Skipping duplicate event: {event_id}")
            return
        
        await self.handle(event)
        await self.processed.setex(event_id, self.ttl, "1")
    
    async def handle(self, event: dict):
        # Override in subclass
        pass
