"""Main application orchestrator for news scraper."""
import asyncio
import logging
import os
import signal
import sys
import time
import yaml
from typing import List, Dict, Any

# Add current directory to path for imports
sys.path.insert(0, os.path.dirname(__file__))

from models.article import Article
from core.crypto_detector import CryptoDetector
from core.sentiment import SentimentAnalyzer
from core.kafka_producer import KafkaProducer
from sources.base_source import BaseSource
from sources.rss_source import RSSSource
from sources.twitter_source import TwitterSource
from sources.reddit_source import RedditSource
from sources.bloomberg_source import BloombergSource


logger = logging.getLogger(__name__)

# Source type mapping
SOURCE_CLASSES = {
    "rss": RSSSource,
    "twitter": TwitterSource,
    "reddit": RedditSource,
    "bloomberg": BloombergSource,
}


class NewsScraperApp:
    """Main application orchestrator."""

    def __init__(self, config_path: str = "config.yaml"):
        """Initialize application."""
        self.running = False
        self.config = self._load_config(config_path)
        self._setup_logging()

        # Initialize components
        self.crypto_detector = CryptoDetector(self.config["crypto_symbols"])
        self.sentiment_analyzer = SentimentAnalyzer(
            enabled=self.config["sentiment"]["enabled"]
        )
        self.kafka_producer = KafkaProducer(
            brokers=self.config["kafka"]["brokers"],
            topic=self.config["kafka"]["topic"]
        )

        # Initialize sources
        self.sources = self._initialize_sources()
        self.interval = self.config["scraping"]["interval_seconds"]

        logger.info("=" * 60)
        logger.info("News Scraper initialized")
        logger.info(f"Active sources: {[s.name for s in self.sources if s.enabled]}")
        logger.info(f"POC stubs: {[s.name for s in self.sources if not s.enabled]}")
        logger.info(f"Kafka: {self.config['kafka']['brokers']} → {self.config['kafka']['topic']}")
        logger.info("=" * 60)

    def _load_config(self, config_path: str) -> Dict[str, Any]:
        """Load configuration from YAML file."""
        with open(config_path, "r") as f:
            config = yaml.safe_load(f)

        # Expand environment variables
        for source in config.get("sources", []):
            for key, value in source.items():
                if isinstance(value, str) and value.startswith("${") and value.endswith("}"):
                    env_var = value[2:-1]
                    source[key] = os.getenv(env_var, "")

        return config

    def _setup_logging(self):
        """Configure logging."""
        level = self.config["logging"]["level"]
        format_str = self.config["logging"]["format"]

        logging.basicConfig(
            level=getattr(logging, level),
            format=format_str,
            handlers=[logging.StreamHandler(sys.stdout)]
        )

    def _initialize_sources(self) -> List[BaseSource]:
        """Initialize all configured sources."""
        sources = []

        for source_config in self.config.get("sources", []):
            source_type = source_config.get("type")
            source_class = SOURCE_CLASSES.get(source_type)

            if not source_class:
                logger.warning(f"Unknown source type: {source_type}")
                continue

            try:
                source = source_class(source_config)
                sources.append(source)
                logger.info(f"Initialized: {source}")
            except Exception as e:
                logger.error(f"Failed to initialize {source_type}: {e}")

        # Sort by priority
        sources.sort(key=lambda s: s.priority)
        return sources

    async def fetch_all(self) -> List[Article]:
        """Fetch articles from all enabled sources."""
        all_articles = []
        active_sources = [s for s in self.sources if s.enabled and s.is_available()]

        if not active_sources:
            logger.warning("No active sources available")
            return []

        logger.info(f"Fetching from {len(active_sources)} sources...")

        # Fetch concurrently
        tasks = [source.fetch_articles() for source in active_sources]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Collect results
        for source, result in zip(active_sources, results):
            if isinstance(result, Exception):
                logger.error(f"{source.name} failed: {result}")
            elif isinstance(result, list):
                all_articles.extend(result)

        logger.info(f"Fetched {len(all_articles)} total articles")
        return all_articles

    def process_articles(self, articles: List[Article]) -> List[Article]:
        """Process articles: detect symbols and analyze sentiment."""
        processed = []

        for article in articles:
            try:
                # Detect crypto symbols
                article.symbols = self.crypto_detector.detect_in_article(
                    article.title, article.content or ""
                )

                # Analyze sentiment
                article.sentiment_score = self.sentiment_analyzer.analyze_article(
                    article.title, article.content or ""
                )

                # Only keep articles with detected symbols
                if article.symbols:
                    processed.append(article)

            except Exception as e:
                logger.error(f"Failed to process article: {e}")

        logger.info(f"Processed {len(processed)}/{len(articles)} articles (with symbols)")
        return processed

    def deduplicate(self, articles: List[Article]) -> List[Article]:
        """Remove duplicate articles based on URL."""
        seen = set()
        unique = []

        for article in articles:
            if article.url not in seen:
                unique.append(article)
                seen.add(article.url)

        if len(articles) != len(unique):
            logger.info(f"Removed {len(articles) - len(unique)} duplicates")

        return unique

    async def run_once(self):
        """Run one scraping cycle."""
        try:
            # Fetch articles
            articles = await self.fetch_all()

            if not articles:
                logger.info("No articles fetched this cycle")
                return

            # Deduplicate
            articles = self.deduplicate(articles)

            # Process (detect symbols, sentiment)
            articles = self.process_articles(articles)

            if not articles:
                logger.info("No articles with crypto symbols found")
                return

            # Publish to Kafka
            published = self.kafka_producer.publish_batch(articles)
            logger.info(f"✅ Published {published} articles to Kafka")

            # Log sample
            if articles:
                sample = articles[0]
                logger.info(f"Sample: {sample}")

        except Exception as e:
            logger.error(f"Error in scraping cycle: {e}", exc_info=True)

    async def run(self):
        """Run main loop."""
        self.running = True
        logger.info(f"Starting main loop (interval: {self.interval}s)...")

        # Check Kafka connection
        if not self.kafka_producer.check_connection():
            logger.error("Kafka not available, exiting...")
            return

        cycle = 0
        while self.running:
            cycle += 1
            logger.info(f"\n{'=' * 60}")
            logger.info(f"Cycle {cycle} - {time.strftime('%Y-%m-%d %H:%M:%S')}")
            logger.info(f"{'=' * 60}")

            await self.run_once()

            if self.running:
                logger.info(f"Sleeping {self.interval}s until next cycle...")
                await asyncio.sleep(self.interval)

    def stop(self):
        """Stop the application."""
        logger.info("Stopping application...")
        self.running = False
        self.kafka_producer.close()


def signal_handler(sig, frame, app):
    """Handle shutdown signals."""
    logger.info(f"Received signal {sig}, shutting down...")
    app.stop()
    sys.exit(0)


async def main():
    """Main entry point."""
    try:
        app = NewsScraperApp()

        # Setup signal handlers
        signal.signal(signal.SIGINT, lambda s, f: signal_handler(s, f, app))
        signal.signal(signal.SIGTERM, lambda s, f: signal_handler(s, f, app))

        await app.run()

    except Exception as e:
        logger.error(f"Fatal error: {e}", exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
