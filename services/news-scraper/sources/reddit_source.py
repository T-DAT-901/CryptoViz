"""Reddit news source (POC stub)."""
import logging
from typing import List, Dict, Any
from sources.base_source import BaseSource
from models.article import Article


logger = logging.getLogger(__name__)


class RedditSource(BaseSource):
    """
    Reddit API implementation (POC stub).

    This is a stub to demonstrate how to add Reddit as a news source.
    In production, this would use PRAW to fetch posts from crypto subreddits.

    To implement:
    1. pip install praw
    2. Get Reddit API credentials (client_id, client_secret)
    3. Implement fetch_articles() using PRAW
    """

    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.client_id = config.get("client_id", "")
        self.client_secret = config.get("client_secret", "")
        self.subreddits = config.get("subreddits", [])

    async def fetch_articles(self) -> List[Article]:
        """
        Fetch posts from configured subreddits.

        TODO: Implement using PRAW
        - Initialize praw.Reddit with credentials
        - Fetch hot/new posts from self.subreddits
        - Filter by flair (e.g., "News", "Discussion")
        - Convert posts to Article format
        - Handle rate limiting
        """
        logger.info(f"{self.name}: POC mode - Reddit integration not yet implemented")
        logger.info(f"Would fetch from subreddits: {', '.join(self.subreddits)}")
        return []

    def is_available(self) -> bool:
        """Check if Reddit API credentials are configured."""
        return bool(self.client_id and self.client_secret and self.subreddits)
