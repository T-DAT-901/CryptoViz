# Fonction de scraping Yahoo Finance
import feedparser

def fetch_articles():
    """
    Récupère les articles du flux RSS crypto de Yahoo Finance
    """
    url = "https://finance.yahoo.com/rss/cryptocurrencies"
    feed = feedparser.parse(url)

    news_items = []
    for entry in feed.entries:
        title = entry.title
        link = entry.link
        published = entry.published
        news_items.append({"title": title, "link": link, "published": published})
    return news_items

