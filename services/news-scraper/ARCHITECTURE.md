# News Scraper - Scalable Architecture

## 📐 Architecture Overview

### Current Issues
- ❌ Monolithic `scraper.py` (172 lines, all in one file)
- ❌ Hardcoded RSS feeds
- ❌ No abstraction for adding new sources
- ❌ No Kafka integration
- ❌ Placeholder sentiment analysis
- ❌ JSON file storage only

### New Architecture (Scalable)

```
services/news-scraper/
├── app.py                     # Main orchestrator
├── config.yaml                # Source configuration
├── requirements.txt
├── Dockerfile
│
├── core/
│   ├── __init__.py
│   ├── kafka_producer.py     # Kafka integration
│   ├── sentiment.py           # VADER sentiment analysis
│   └── crypto_detector.py     # Symbol detection
│
├── sources/
│   ├── __init__.py
│   ├── base_source.py         # Abstract base class
│   ├── rss_source.py          # RSS implementation
│   ├── twitter_source.py      # Twitter API (stub/POC)
│   ├── reddit_source.py       # Reddit API (stub/POC)
│   └── bloomberg_source.py    # Bloomberg scraper (stub/POC)
│
├── models/
│   ├── __init__.py
│   └── article.py             # Article data model
│
└── utils/
    ├── __init__.py
    ├── logger.py              # Structured logging
    └── config_loader.py       # YAML config loader
```

## 🏗️ Design Patterns

### 1. **Abstract Factory Pattern** (Base Source)

```python
from abc import ABC, abstractmethod
from typing import List, Dict, Any

class BaseSource(ABC):
    """Abstract base class for all news sources"""

    def __init__(self, config: Dict[str, Any]):
        self.name = config.get("name", "Unknown")
        self.enabled = config.get("enabled", True)
        self.priority = config.get("priority", 1)

    @abstractmethod
    async def fetch_articles(self) -> List[Article]:
        """Fetch articles from the source"""
        pass

    @abstractmethod
    def is_available(self) -> bool:
        """Check if source is available"""
        pass

    def get_metadata(self) -> Dict[str, Any]:
        """Return source metadata"""
        return {
            "name": self.name,
            "enabled": self.enabled,
            "priority": self.priority
        }
```

### 2. **Strategy Pattern** (Source Manager)

```python
class SourceManager:
    """Orchestrates multiple sources"""

    def __init__(self, sources: List[BaseSource]):
        self.sources = sorted(sources, key=lambda s: s.priority)

    async def fetch_all(self) -> List[Article]:
        """Fetch from all enabled sources concurrently"""
        tasks = [
            source.fetch_articles()
            for source in self.sources
            if source.enabled and source.is_available()
        ]
        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Flatten and deduplicate
        articles = []
        seen_urls = set()
        for result in results:
            if isinstance(result, list):
                for article in result:
                    if article.url not in seen_urls:
                        articles.append(article)
                        seen_urls.add(article.url)
        return articles
```

### 3. **Data Model** (Article)

```python
from dataclasses import dataclass
from datetime import datetime
from typing import List, Optional

@dataclass
class Article:
    title: str
    url: str
    source: str
    symbols: List[str]
    published_at: datetime
    content: Optional[str] = None
    sentiment_score: Optional[float] = None

    def to_kafka_message(self) -> Dict[str, Any]:
        """Convert to Kafka message format matching NewsHandler"""
        return {
            "type": "news",
            "time": int(self.published_at.timestamp() * 1000),
            "source": self.source,
            "url": self.url,
            "title": self.title,
            "content": self.content or "",
            "sentiment_score": self.sentiment_score,
            "symbols": self.symbols
        }
```

## 📝 Configuration (config.yaml)

```yaml
# Kafka Configuration
kafka:
  brokers: "kafka:9092"
  topic: "crypto.news"
  client_id: "news-scraper"

# Crypto Detection
crypto_symbols:
  - name: "btc"
    keywords: ["bitcoin", "btc"]
  - name: "eth"
    keywords: ["ethereum", "eth"]
  - name: "xrp"
    keywords: ["ripple", "xrp"]
  - name: "ada"
    keywords: ["cardano", "ada"]
  - name: "sol"
    keywords: ["solana", "sol"]
  - name: "doge"
    keywords: ["dogecoin", "doge"]
  - name: "bnb"
    keywords: ["binance", "bnb"]
  - name: "ltc"
    keywords: ["litecoin", "ltc"]

# News Sources
sources:
  # RSS Feeds (Active)
  - type: "rss"
    name: "CoinDesk"
    enabled: true
    priority: 1
    feeds:
      - "https://www.coindesk.com/arc/outboundfeeds/rss/?outputType=xml"
      - "https://www.coindesk.com/tag/bitcoin/feed/"
      - "https://www.coindesk.com/tag/ethereum/feed/"
      - "https://www.coindesk.com/markets/feed/"

  - type: "rss"
    name: "CoinTelegraph"
    enabled: true
    priority: 2
    feeds:
      - "https://cointelegraph.com/rss"
      - "https://cointelegraph.com/rss/tag/bitcoin"
      - "https://cointelegraph.com/rss/tag/ethereum"

  # Social Media (POC/Stubs - for demonstration)
  - type: "twitter"
    name: "Twitter"
    enabled: false  # Disabled by default
    priority: 3
    api_key: "${TWITTER_API_KEY}"
    accounts: ["coindesk", "cointelegraph", "VitalikButerin"]

  - type: "reddit"
    name: "Reddit"
    enabled: false
    priority: 4
    client_id: "${REDDIT_CLIENT_ID}"
    client_secret: "${REDDIT_CLIENT_SECRET}"
    subreddits: ["cryptocurrency", "bitcoin", "ethereum"]

  # Financial News (POC/Stubs)
  - type: "bloomberg"
    name: "Bloomberg"
    enabled: false
    priority: 5
    scrape_url: "https://www.bloomberg.com/crypto"

# Sentiment Analysis
sentiment:
  enabled: true
  provider: "vader"  # Options: vader, textblob, custom

# Scraping Settings
scraping:
  interval_seconds: 60
  max_articles_per_source: 50
  days_back: 5
  user_agent: "Mozilla/5.0 CryptoViz News Scraper"
  timeout_seconds: 10

# Logging
logging:
  level: "INFO"  # DEBUG, INFO, WARNING, ERROR
  format: "json"
  output: "stdout"
```

## 🔌 Source Implementations

### RSS Source (Active)
```python
class RSSSource(BaseSource):
    """RSS feed implementation"""

    def __init__(self, config: Dict[str, Any]):
        super().__init__(config)
        self.feeds = config.get("feeds", [])

    async def fetch_articles(self) -> List[Article]:
        articles = []
        for feed_url in self.feeds:
            try:
                feed = await self._fetch_feed(feed_url)
                articles.extend(self._parse_feed(feed))
            except Exception as e:
                logger.error(f"RSS fetch failed: {feed_url} - {e}")
        return articles
```

### Twitter Source (POC Stub)
```python
class TwitterSource(BaseSource):
    """Twitter API implementation (POC)"""

    async def fetch_articles(self) -> List[Article]:
        # TODO: Implement Twitter API v2
        # For now, return empty to demonstrate architecture
        logger.info("TwitterSource: POC mode - returning empty")
        return []

    def is_available(self) -> bool:
        # Check if API credentials are present
        return bool(self.config.get("api_key"))
```

### Reddit Source (POC Stub)
```python
class RedditSource(BaseSource):
    """Reddit API implementation (POC)"""

    async def fetch_articles(self) -> List[Article]:
        # TODO: Implement Reddit PRAW
        logger.info("RedditSource: POC mode - returning empty")
        return []

    def is_available(self) -> bool:
        return bool(self.config.get("client_id"))
```

### Bloomberg Source (POC Stub)
```python
class BloombergSource(BaseSource):
    """Bloomberg scraper (POC)"""

    async def fetch_articles(self) -> List[Article]:
        # TODO: Implement Bloomberg scraping
        logger.info("BloombergSource: POC mode - returning empty")
        return []
```

## 📊 Data Flow

```
┌─────────────────────────────────────────────────────┐
│                   Source Manager                     │
│                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │   RSS    │  │ Twitter  │  │  Reddit  │ ...      │
│  │  Source  │  │  Source  │  │  Source  │          │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘          │
│       │             │              │                │
│       └─────────────┴──────────────┘                │
│                     │                               │
│                     ▼                               │
│              [Deduplication]                        │
└─────────────────────┬───────────────────────────────┘
                      │
                      ▼
          ┌───────────────────────┐
          │   Crypto Detector     │  ← Detect symbols
          └───────────┬───────────┘
                      │
                      ▼
          ┌───────────────────────┐
          │  Sentiment Analyzer   │  ← VADER analysis
          └───────────┬───────────┘
                      │
                      ▼
          ┌───────────────────────┐
          │   Kafka Producer      │  → crypto.news topic
          └───────────┬───────────┘
                      │
                      ▼
          ┌───────────────────────┐
          │     Backend-Go        │  ← NewsHandler
          │   (NewsConsumer)      │
          └───────────────────────┘
```

## 🚀 Benefits

### ✅ **Easy to Add Sources**
Adding a new source is as simple as:
1. Create `new_source.py` inheriting from `BaseSource`
2. Implement `fetch_articles()` and `is_available()`
3. Add configuration to `config.yaml`
4. Source Manager auto-discovers and loads it

### ✅ **Priority & Control**
- Sources have priority levels (1 = highest)
- Enable/disable sources via config
- Graceful degradation if source fails

### ✅ **Scalability**
- Async/await for concurrent fetching
- Each source runs independently
- Easy to add rate limiting per source

### ✅ **Testability**
- Mock sources for testing
- Unit test each source independently
- Integration tests with Kafka

### ✅ **Maintainability**
- Single Responsibility Principle
- Clear separation of concerns
- Easy to debug individual sources

## 📈 Demonstration of Scalability (POC)

**Before** (Monolithic):
```python
# To add Twitter: Modify scraper.py, add 100+ lines
# To add Reddit: Modify scraper.py, add 100+ lines
# Sources tightly coupled, hard to test
```

**After** (Modular):
```python
# To add Twitter: Create twitter_source.py (50 lines)
# To add Reddit: Create reddit_source.py (50 lines)
# Add 3 lines to config.yaml
# Zero changes to existing code
```

## 🎯 Implementation Plan

### Phase 1: Core Refactor (Day 1)
- [x] Design architecture
- [ ] Create base classes and interfaces
- [ ] Implement Article data model
- [ ] Create Source Manager

### Phase 2: RSS Migration (Day 1)
- [ ] Refactor existing RSS code to RSSSource
- [ ] Implement crypto detector
- [ ] Implement VADER sentiment
- [ ] Test with existing data

### Phase 3: Kafka Integration (Day 2)
- [ ] Implement Kafka Producer
- [ ] Message format matching NewsHandler
- [ ] Error handling and retry logic
- [ ] End-to-end testing

### Phase 4: POC Stubs (Day 2)
- [ ] Create Twitter source stub
- [ ] Create Reddit source stub
- [ ] Create Bloomberg source stub
- [ ] Add all to config.yaml with enabled=false

### Phase 5: Documentation & Demo (Day 3)
- [ ] Complete architecture docs
- [ ] Demo video showing scalability
- [ ] Performance metrics
- [ ] Future roadmap

---

**Status:** 🟡 Architecture designed, ready for implementation
**Timeline:** 2-3 days
**LOC Estimate:** ~800 lines (vs 172 monolithic)
