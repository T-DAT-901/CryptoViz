import json
import os
from datetime import datetime

ARTICLES_FILE = "articles.json"

def load_articles():
    if not os.path.exists(ARTICLES_FILE):
        return []
    try:
        with open(ARTICLES_FILE, "r", encoding="utf-8") as f:
            articles = json.load(f)
    except json.JSONDecodeError:
        return []

    for a in articles:
        if a.get("published_dt"):
            try:
                a["published_dt"] = datetime.fromisoformat(a["published_dt"])
            except Exception:
                a["published_dt"] = None

    return articles

def save_articles(new_articles):
    all_articles = load_articles()

    for a in new_articles:
        if isinstance(a.get("published_dt"), datetime):
            a["published_dt"] = a["published_dt"].isoformat()

    all_articles.extend(new_articles)

    seen_links = set()
    unique_articles = []
    for a in all_articles:
        if isinstance(a.get("published_dt"), datetime):
            a["published_dt"] = a["published_dt"].isoformat()

        if a["link"] not in seen_links:
            unique_articles.append(a)
            seen_links.add(a["link"])

    with open(ARTICLES_FILE, "w", encoding="utf-8") as f:
        json.dump(unique_articles, f, ensure_ascii=False, indent=2)
