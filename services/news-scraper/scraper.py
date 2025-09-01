# Fonction de scraping Yahoo Finance

import feedparser
from nltk.sentiment import SentimentIntensityAnalyzer

analyzer = SentimentIntensityAnalyzer()

def fetch_articles(count=20):
    """
    Récupère les articles récents via le flux RSS de Yahoo Finance.
    """
    rss_url = "https://finance.yahoo.com/rss/"
    feed = feedparser.parse(rss_url)
    articles = []

    for entry in feed.entries[:count]:
        sentiment_scores = analyzer.polarity_scores(entry.title)
        compound = sentiment_scores["compound"]

        if compound >= 0.05:
            sentiment = "positif"
        elif compound <= -0.05:
            sentiment = "négatif"
        else:
            sentiment = "neutre"

        articles.append({
            "title": entry.title,
            "link": entry.link,
            "published": entry.published,
            "sentiment": sentiment
        })

    return articles

