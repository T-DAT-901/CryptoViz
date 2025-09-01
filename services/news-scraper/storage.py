# Sauvegarde des données

import pandas as pd
import os

STORAGE_FILE = "articles.csv"


def load_articles():
    """Charge tous les articles stockés dans un DataFrame."""
    if os.path.exists(STORAGE_FILE):
        return pd.read_csv(STORAGE_FILE)
    return pd.DataFrame(columns=["title", "link", "published"])


def save_articles(new_articles):
    """Ajoute de nouveaux articles au fichier CSV en évitant les doublons."""
    df_existing = load_articles()
    df_new = pd.DataFrame(new_articles)

    # Évite les doublons basés sur le lien
    df_combined = pd.concat([df_existing, df_new]).drop_duplicates(subset="link", keep="last")

    df_combined.to_csv(STORAGE_FILE, index=False)
    print(f"{len(df_combined) - len(df_existing)} nouveaux articles sauvegardés.")
