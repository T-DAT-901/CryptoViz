"""Article data model matching backend-go NewsHandler format."""
from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Optional, Dict, Any


@dataclass
class Article:
    """
    Article data model for news sources.

    Matches the message format expected by backend-go NewsHandler:
    {
        "type": "news",
        "time": 1700478621000,  # Unix timestamp ms
        "source": "CoinDesk",
        "url": "https://...",
        "title": "Bitcoin hits new high",
        "content": "...",
        "sentiment_score": 0.75,  # -1 to +1
        "symbols": ["btc", "eth"]
    }
    """

    title: str
    url: str
    source: str
    published_at: datetime
    symbols: List[str] = field(default_factory=list)
    content: Optional[str] = None
    sentiment_score: Optional[float] = None

    def to_kafka_message(self) -> Dict[str, Any]:
        """
        Convert article to Kafka message format for crypto.news topic.

        Returns:
            Dict matching backend-go NewsHandler NewsMessage struct
        """
        return {
            "type": "news",
            "time": int(self.published_at.timestamp() * 1000),  # Unix ms
            "source": self.source,
            "url": self.url,
            "title": self.title,
            "content": self.content or "",
            "sentiment_score": self.sentiment_score,
            "symbols": self.symbols,
        }

    def __hash__(self):
        """Hash based on URL for deduplication."""
        return hash(self.url)

    def __eq__(self, other):
        """Equality based on URL."""
        if not isinstance(other, Article):
            return False
        return self.url == other.url

    def __repr__(self):
        """String representation."""
        symbols_str = ", ".join(self.symbols) if self.symbols else "none"
        sentiment_str = f"{self.sentiment_score:+.2f}" if self.sentiment_score else "N/A"
        return (
            f"Article(source='{self.source}', "
            f"symbols=[{symbols_str}], "
            f"sentiment={sentiment_str}, "
            f"title='{self.title[:50]}...')"
        )
