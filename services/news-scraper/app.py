import time
from scraper import fetch_articles
from storage import save_articles, load_articles

def main():
    existing_articles = load_articles()
    seen_links = {article["link"] for article in existing_articles}

    while True:
        print("----- Récupération des articles -----")
        articles = fetch_articles()

        new_articles = [a for a in articles if a["link"] not in seen_links]

        if new_articles:
            save_articles(new_articles)
            seen_links.update(a["link"] for a in new_articles)
        else:
            print("X Aucun nouvel article trouvé X")

        time.sleep(10)

if __name__ == "__main__":
    main()
