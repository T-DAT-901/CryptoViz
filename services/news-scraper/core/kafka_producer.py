"""Kafka producer for news articles."""
import json
import logging
from typing import List
from confluent_kafka import Producer
from confluent_kafka.admin import AdminClient
from models.article import Article


logger = logging.getLogger(__name__)


class KafkaProducer:
    """Kafka producer for publishing news articles."""

    def __init__(self, brokers: str, topic: str):
        """
        Initialize Kafka producer.

        Args:
            brokers: Kafka broker addresses (comma-separated)
            topic: Topic name (e.g., "crypto.news")
        """
        self.brokers = brokers
        self.topic = topic
        self.producer = Producer({
            'bootstrap.servers': brokers,
            'client.id': 'news-scraper',
            'compression.type': 'lz4',
            'linger.ms': 100,  # Batch messages for 100ms
            'batch.size': 16384,  # 16KB batches
        })
        logger.info(f"Kafka producer initialized: topic={topic}, brokers={brokers}")

    def publish(self, article: Article) -> bool:
        """
        Publish single article to Kafka.

        Args:
            article: Article to publish

        Returns:
            True if published successfully
        """
        try:
            message = article.to_kafka_message()
            value = json.dumps(message).encode('utf-8')

            # Asynchronous produce with callback
            self.producer.produce(
                self.topic,
                value=value,
                callback=self._delivery_callback
            )
            self.producer.poll(0)  # Trigger callbacks
            return True

        except Exception as e:
            logger.error(f"Failed to publish article: {e}", exc_info=True)
            return False

    def publish_batch(self, articles: List[Article]) -> int:
        """
        Publish multiple articles to Kafka.

        Args:
            articles: List of articles

        Returns:
            Number of articles successfully queued
        """
        success_count = 0

        for article in articles:
            if self.publish(article):
                success_count += 1

        # Wait for all messages to be delivered
        self.producer.flush(timeout=10)
        logger.info(f"Published {success_count}/{len(articles)} articles to Kafka")
        return success_count

    def _delivery_callback(self, err, msg):
        """Callback for message delivery confirmation."""
        if err:
            logger.error(f"Message delivery failed: {err}")
        else:
            logger.debug(f"Message delivered to {msg.topic()} [{msg.partition()}] @ offset {msg.offset()}")

    def close(self):
        """Close producer and flush remaining messages."""
        logger.info("Closing Kafka producer...")
        self.producer.flush(timeout=30)
        logger.info("Kafka producer closed")

    def check_connection(self) -> bool:
        """Check if Kafka is reachable."""
        try:
            admin = AdminClient({'bootstrap.servers': self.brokers})
            metadata = admin.list_topics(timeout=5)
            logger.info(f"Kafka connection OK: {len(metadata.topics)} topics available")
            return True
        except Exception as e:
            logger.error(f"Kafka connection failed: {e}")
            return False
