"""RSS feed news source."""
import logging
from typing import List, Dict, Any
from datetime import datetime, timedelta
import feedparser
import aiohttp
from sources.base_source import BaseSource
from models.article import Article


logger = logging.getLogger(__name__)


class RSSSource(BaseSource):
    """RSS feed implementation."""

    def __init__(self, config: Dict[str, Any]):
        """Initialize RSS source."""
        super().__init__(config)
        self.feeds = config.get("feeds", [])
        self.days_back = config.get("days_back", 5)
        self.user_agent = config.get("user_agent", "Mozilla/5.0 CryptoViz News Scraper")

    async def fetch_articles(self) -> List[Article]:
        """Fetch articles from all configured RSS feeds."""
        all_articles = []

        for feed_url in self.feeds:
            try:
                articles = await self._fetch_feed(feed_url)
                all_articles.extend(articles)
                logger.info(f"Fetched {len(articles)} articles from {feed_url}")
            except Exception as e:
                logger.error(f"Failed to fetch RSS feed {feed_url}: {e}", exc_info=True)

        # Filter by days_back
        cutoff_date = datetime.utcnow() - timedelta(days=self.days_back)
        recent_articles = [a for a in all_articles if a.published_at >= cutoff_date]

        logger.info(f"{self.name}: Fetched {len(recent_articles)}/{len(all_articles)} recent articles")
        return recent_articles

    async def _fetch_feed(self, feed_url: str) -> List[Article]:
        """Fetch and parse single RSS feed."""
        articles = []

        try:
            # Fetch feed with aiohttp
            headers = {
                "User-Agent": self.user_agent,
                "Accept": "application/xml, text/xml"
            }

            async with aiohttp.ClientSession() as session:
                async with session.get(feed_url, headers=headers, timeout=10) as response:
                    response.raise_for_status()
                    content = await response.text()

            # Parse with feedparser
            feed = feedparser.parse(content)

            for entry in feed.entries:
                article = self._parse_entry(entry)
                if article:
                    articles.append(article)

        except Exception as e:
            logger.error(f"Error fetching feed {feed_url}: {e}")
            raise

        return articles

    def _parse_entry(self, entry) -> Article:
        """Parse single RSS entry into Article."""
        try:
            title = entry.get("title", "").strip()
            url = entry.get("link", "")
            summary = entry.get("summary", "").strip()

            # Parse published date
            if hasattr(entry, 'published_parsed') and entry.published_parsed:
                published_at = datetime(*entry.published_parsed[:6])
            else:
                published_at = datetime.utcnow()

            # Create article (symbols and sentiment will be added later)
            article = Article(
                title=title,
                url=url,
                source=self.name,
                published_at=published_at,
                content=summary,
                symbols=[],  # Will be populated by CryptoDetector
                sentiment_score=None  # Will be populated by SentimentAnalyzer
            )

            return article

        except Exception as e:
            logger.error(f"Failed to parse RSS entry: {e}")
            return None

    def is_available(self) -> bool:
        """Check if RSS feeds are configured."""
        return bool(self.feeds)
