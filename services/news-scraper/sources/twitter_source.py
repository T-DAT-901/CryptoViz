"""Twitter news source (POC stub)."""
import logging
from typing import List, Dict, Any
from sources.base_source import BaseSource
from models.article import Article


logger = logging.getLogger(__name__)


class TwitterSource(BaseSource):
    """
    Twitter API implementation (POC stub).

    This is a stub to demonstrate how to add Twitter as a news source.
    In production, this would use Twitter API v2 to fetch tweets from
    crypto-related accounts.

    To implement:
    1. pip install tweepy
    2. Get Twitter API credentials
    3. Implement fetch_articles() using Tweepy client
    """

    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.api_key = config.get("api_key", "")
        self.accounts = config.get("accounts", [])

    async def fetch_articles(self) -> List[Article]:
        """
        Fetch tweets from configured accounts.

        TODO: Implement using Twitter API v2
        - Use tweepy.Client with bearer_token
        - Fetch recent tweets from self.accounts
        - Convert tweets to Article format
        - Handle rate limiting
        """
        logger.info(f"{self.name}: POC mode - Twitter integration not yet implemented")
        logger.info(f"Would fetch tweets from: {', '.join(self.accounts)}")
        return []

    def is_available(self) -> bool:
        """Check if Twitter API credentials are configured."""
        return bool(self.api_key and self.accounts)
