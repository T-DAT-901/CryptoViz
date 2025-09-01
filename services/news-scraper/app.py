# Point d'entrée qui orchestre tout

import time
from scraper import fetch_articles
from storage import save_articles, load_articles

def main():
    seen_links = set(load_articles()["link"])  # Charge les articles déjà connus

    while True:
        print("🔍 Récupération des articles...")
        articles = fetch_articles(count=20)

        # Filtre uniquement les nouveaux
        new_articles = [a for a in articles if a["link"] not in seen_links]

        if new_articles:
            save_articles(new_articles)
            # Ajoute au set pour éviter doublons pendant le même run
            seen_links.update(a["link"] for a in new_articles)
        else:
            print("Aucun nouvel article trouvé.")

        time.sleep(120)  # Attente 2 min avant de relancer

if __name__ == "__main__":
    main()

