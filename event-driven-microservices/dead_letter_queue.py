"""Dead letter queue with automatic retry."""
import json

class DeadLetterQueue:
    def __init__(self, producer, topic: str, max_retries: int = 3):
        self.producer = producer
        self.dlq_topic = f"{topic}.dlq"
        self.retry_topic = f"{topic}.retry"
        self.max_retries = max_retries
    
    async def handle_failure(self, event: dict, error: Exception):
        retry_count = event.get("retry_count", 0)
        if retry_count < self.max_retries:
            event["retry_count"] = retry_count + 1
            await self.producer.send(self.retry_topic, event)
            print(f"Retry {retry_count + 1}/{self.max_retries}")
        else:
            await self.producer.send(self.dlq_topic, event)
            print(f"Sent to DLQ: {event.get('id')}")
