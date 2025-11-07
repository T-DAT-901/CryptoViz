# coindesk_scraper.py

import feedparser
from datetime import datetime

# Dictionnaire pour détecter la crypto concernée // À voir pour en rajouter par la suite
CRYPTO_KEYWORDS = {
    "btc": ["bitcoin", "btc"],
    "eth": ["ethereum", "eth"],
    "xrp": ["ripple", "xrp"],
    "ada": ["cardano", "ada"],
    "sol": ["solana", "sol"],
    "doge": ["dogecoin", "doge"],
    "bnb": ["binance", "bnb"],
    "ltc": ["litecoin", "ltc"]
}


def detect_crypto(text: str) -> str:
    """
    Détecte la crypto concernée (btc, eth, etc.) dans un texte.
    Retourne 'multi' si plusieurs cryptos sont mentionnées, ou 'other' si aucune.
    """
    text_lower = text.lower()
    found = [symbol for symbol, keywords in CRYPTO_KEYWORDS.items()
             if any(k in text_lower for k in keywords)]

    if len(found) == 1:
        return found[0]
    elif len(found) > 1:
        return "multi"
    else:
        return "other"


def fetch_articles():
    """
    Récupère les articles du flux RSS de CoinDesk
    """
    url = "https://www.coindesk.com/arc/outboundfeeds/rss/"
    feed = feedparser.parse(url)

    news_items = []

    for entry in feed.entries:
        title = entry.get("title", "Sans titre").strip()
        link = entry.get("link", "")
        summary = entry.get("summary", "").strip()
        published = entry.get("published", "")

        # Conversion de la date en ISO
        try:
            published_dt = datetime(*entry.published_parsed[:6])
            published_str = published_dt.isoformat()
        except Exception:
            published_str = published

        # Détection de la crypto concernée
        money = detect_crypto(f"{title} {summary}")

        news_items.append({
            "title": title,
            "summary": summary,
            "link": link,
            "published": published_str,
            "source": "CoinDesk",
            "money": money
        })

    return news_items


# Test rapide
if __name__ == "__main__":
    articles = fetch_articles()
    for a in articles[:10]:
        print(f"{a['money'].upper():<5} | {a['published']} | {a['title']}")
