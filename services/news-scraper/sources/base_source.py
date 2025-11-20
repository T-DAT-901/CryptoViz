"""Abstract base class for all news sources."""
from abc import ABC, abstractmethod
from typing import List, Dict, Any
from models.article import Article


class BaseSource(ABC):
    """Abstract base class for all news sources."""

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize source with configuration.

        Args:
            config: Source configuration from config.yaml
        """
        self.name = config.get("name", "Unknown")
        self.enabled = config.get("enabled", True)
        self.priority = config.get("priority", 1)
        self.config = config

    @abstractmethod
    async def fetch_articles(self) -> List[Article]:
        """
        Fetch articles from the source.

        Returns:
            List of Article objects

        Raises:
            Exception: If fetching fails
        """
        pass

    @abstractmethod
    def is_available(self) -> bool:
        """
        Check if source is available and properly configured.

        Returns:
            True if source can be used, False otherwise
        """
        pass

    def get_metadata(self) -> Dict[str, Any]:
        """
        Get source metadata.

        Returns:
            Dict with source information
        """
        return {
            "name": self.name,
            "enabled": self.enabled,
            "priority": self.priority,
            "available": self.is_available(),
        }

    def __repr__(self):
        """String representation."""
        status = "enabled" if self.enabled else "disabled"
        available = "available" if self.is_available() else "unavailable"
        return f"{self.__class__.__name__}(name='{self.name}', {status}, {available}, priority={self.priority})"
