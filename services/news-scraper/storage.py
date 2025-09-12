import json
import os

STORAGE_FILE = "articles.json"


def load_articles():
    """Charge tous les articles stockés dans une liste de dictionnaires."""
    if os.path.exists(STORAGE_FILE):
        with open(STORAGE_FILE, "r", encoding="utf-8") as f:
            return json.load(f)
    return []


def save_articles(new_articles):
    """Ajoute de nouveaux articles au fichier JSON en évitant les doublons basés sur le lien."""
    existing_articles = load_articles()

    existing_links = {article["link"] for article in existing_articles}

    filtered_new = [article for article in new_articles if article["link"] not in existing_links]

    combined_articles = existing_articles + filtered_new

    # Sauvegarde dans le fichier JSON
    with open(STORAGE_FILE, "w", encoding="utf-8") as f:
        json.dump(combined_articles, f, ensure_ascii=False, indent=4)

    print(f"{len(filtered_new)} nouveaux articles sauvegardés.")
