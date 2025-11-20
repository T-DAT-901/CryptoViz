import feedparser
import requests
from bs4 import BeautifulSoup
from datetime import datetime, timedelta
from textblob import TextBlob
import pandas as pd

# --------------------------
# Configuration
# --------------------------
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

DAYS_BACK = 5  # Nombre de jours d'historique à récupérer

RSS_FEEDS = [
    "https://www.coindesk.com/arc/outboundfeeds/rss/?outputType=xml",
    "https://www.coindesk.com/tag/bitcoin/feed/",
    "https://www.coindesk.com/tag/ethereum/feed/",
    "https://www.coindesk.com/markets/feed/"
]

# Détection des cryptos
def detect_crypto(text: str) -> str:
    text_lower = text.lower()
    found = [symbol for symbol, keywords in CRYPTO_KEYWORDS.items()
             if any(k in text_lower for k in keywords)]
    if len(found) == 1:
        return found[0]
    elif len(found) > 1:
        return "multi"
    else:
        return "other"

# Récupération via RSS
def fetch_articles_rss():
    all_articles = []
    headers = {
        "User-Agent": "Mozilla/5.0",
        "Accept": "application/xml, text/xml"
    }

    for url in RSS_FEEDS:
        try:
            r = requests.get(url, headers=headers, timeout=10)
            r.raise_for_status()
            feed = feedparser.parse(r.text)
            print(f"[DEBUG] RSS feed: {url} - entries: {len(feed.entries)}")

            if not feed.entries:
                continue

            for entry in feed.entries:
                title = entry.get("title", "Sans titre").strip()
                link = entry.get("link", "")
                summary = entry.get("summary", "").strip()
                published = entry.get("published", "")

                try:
                    published_dt = datetime(*entry.published_parsed[:6])
                except Exception:
                    published_dt = None

                money = detect_crypto(f"{title} {summary}")

                all_articles.append({
                    "title": title,
                    "summary": summary,
                    "link": link,
                    "published": published,
                    "published_dt": published_dt,
                    "source": "CoinDesk",
                    "money": money
                })

        except Exception as e:
            print(f"[ERROR] RSS fetch failed for {url}: {e}")

    return all_articles

# Scraper web fallback
def fetch_articles_web():
    base_url = "https://www.coindesk.com/"
    articles = []

    for page in range(1, 10):
        url = base_url if page == 1 else f"{base_url}page/{page}/"
        r = requests.get(url, headers={"User-Agent": "Mozilla/5.0"})
        soup = BeautifulSoup(r.text, "html.parser")

        cards = soup.find_all("div", {"data-qa": "card"})
        if not cards:
            break

        for card in cards:
            link_tag = card.find("a", href=True)
            title_tag = card.find("h3")
            time_tag = card.find("time")

            if not link_tag or not title_tag:
                continue

            link = link_tag["href"]
            if not link.startswith("http"):
                link = "https://www.coindesk.com" + link

            title = title_tag.get_text(strip=True)
            published_dt = None
            published_str = ""

            if time_tag:
                published_str = time_tag.get("datetime", "")
                if published_str:
                    try:
                        published_dt = datetime.fromisoformat(
                            published_str.replace("Z", "+00:00")
                        )
                    except Exception:
                        published_dt = None

            money = detect_crypto(title)

            articles.append({
                "title": title,
                "summary": "",
                "link": link,
                "published": published_str,
                "published_dt": published_dt,
                "source": "CoinDesk",
                "money": money
            })

    return articles

# Filtrage par DAYS_BACK
def filter_articles_by_days(articles):
    cutoff_date = datetime.utcnow() - timedelta(days=DAYS_BACK)
    return [a for a in articles if a["published_dt"] and a["published_dt"] >= cutoff_date]


# Analyse de sentiments
def analyze_sentiment(articles: list) -> list:
    """
    Analyse le sentiment du titre/résumé pour chaque article
    Ajoute les champs 'polarite', 'sentiment' et 'subjectivite'
    """
    analyzed_articles = []

    for article in articles:
        text_to_analyze = (article.get('summary') or article.get('title')).strip()

        # Initialisation par défaut
        article["polarite"] = 0.0
        article["subjectivite"] = 0.0
        article["sentiment"] = "Neutre"

        if not text_to_analyze:
            analyzed_articles.append(article)
            continue

        try:
            blob = TextBlob(text_to_analyze)

            polarite = blob.sentiment.polarity
            subjectivite = blob.sentiment.subjectivity

            if polarite > 0.1:
                sentiment = "Positif"
            elif polarite < -0.1:
                sentiment = "Négatif"
            else:
                sentiment = "Neutre"

            article["polarite"] = polarite
            article["subjectivite"] = subjectivite
            article["sentiment"] = sentiment

        except Exception as e:
            print(f"[ERROR] Sentiment analysis failed for article {article.get('link')}: {e}")
            article["sentiment"] = "Erreur"

        analyzed_articles.append(article)

    return analyzed_articles


# Fonction principale
def fetch_and_analyze_articles():
    """
    Récupère, filtre et analyse le sentiment des articles.
    """
    articles = fetch_articles_rss()
    if not articles:
        articles = fetch_articles_web()

    articles_filtres = filter_articles_by_days(articles)

    if not articles_filtres:
        return []

    articles_analyses = analyze_sentiment(articles_filtres)

    return articles_analyses

# Test manuel
if __name__ == "__main__":
    articles = fetch_and_analyze_articles()
    if not articles:
        print(f"Aucun article trouvé dans les {DAYS_BACK} derniers jours.")
    else:
        for a in articles[:50]:
            pub = a["published"] if a["published"] else "N/A"
            print(f"{a['money'].upper():<5} | {pub} | {a['title']}")
