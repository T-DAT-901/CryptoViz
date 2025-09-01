# Fonction de scraping Yahoo Finance

import feedparser

def fetch_articles(count=20):
    """
    Récupère les articles récents via le flux RSS de Yahoo Finance.
    """
    rss_url = "https://finance.yahoo.com/rss/"
    feed = feedparser.parse(rss_url)
    articles = []

    for entry in feed.entries[:count]:
        articles.append({
            "title": entry.title,
            "link": entry.link,
            "published": entry.published
        })

    return articles

