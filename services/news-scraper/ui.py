# ui.py — Interface graphique des actualités crypto

import tkinter as tk
from storage import load_articles
import webbrowser
from datetime import datetime
from dateutil import parser  # pip install python-dateutil

# Rafraîchissement automatique (en millisecondes)
REFRESH_INTERVAL = 60 * 1000  # 1 minute


def open_link(event, link: str):
    """Ouvre le lien de l'article dans le navigateur par défaut."""
    webbrowser.open(link)


def safe_parse_date(date_str):
    """Essaie de convertir une date texte en objet datetime."""
    try:
        return parser.parse(date_str)
    except Exception:
        return datetime.min


def refresh_articles(content_frame, canvas, status_label=None, crypto_filter=None):
    """Rafraîchit la liste des articles dans l'interface."""
    if status_label:
        status_label.config(text="Actualisation en cours...")
        status_label.update_idletasks()

    # Vide le contenu avant de recharger
    for widget in content_frame.winfo_children():
        widget.destroy()

    # Récupère et trie les articles
    articles = load_articles()
    articles = sorted(articles, key=lambda x: safe_parse_date(x["published"]), reverse=True)

    # Applique un filtre si nécessaire
    if crypto_filter and crypto_filter.get() != "all":
        articles = [a for a in articles if a.get("money") == crypto_filter.get()]

    # Palette de couleurs selon le sentiment
    sentiment_colors = {"positif": "green","négatif": "red","neutre": "gray"}

    # Affiche chaque article
    for article in articles:
        title_label = tk.Label(
            content_frame,
            text=article["title"],
            font=("Arial", 12, "bold"),
            wraplength=600,
            justify="left",
            fg="black",
            cursor="hand2"
        )
        title_label.pack(anchor="w", pady=(0, 2))
        title_label.bind("<Button-1>", lambda e, l=article["link"]: open_link(e, l))

        # Informations additionnelles
        info_text = f"Publié le : {article['published']}"

        if "money" in article:
            info_text += f" | Crypto : {article['money'].upper()}"
        if "sentiment" in article:
            info_text += f" | Sentiment : {article['sentiment']}"
        if "source" in article:
            info_text += f" | Source : {article['source']}"

        color = sentiment_colors.get(article.get("sentiment", "neutre"), "gray")

        info_label = tk.Label(
            content_frame,
            text=info_text,
            font=("Arial", 10),
            fg=color
        )
        info_label.pack(anchor="w", pady=(0, 10))

    # Met à jour la zone de scroll
    content_frame.update_idletasks()
    canvas.config(scrollregion=canvas.bbox("all"))

    # Met à jour
    if status_label:
        now = datetime.now().strftime("%H:%M:%S")
        status_label.config(text=f"Dernière mise à jour : {now}")

    # Prochaine actualisation
    content_frame.after(REFRESH_INTERVAL,lambda: refresh_articles(content_frame, canvas, status_label, crypto_filter))


def main():
    """Lance l'interface principale."""
    root = tk.Tk()
    root.title("Crypto News RSS - Articles sauvegardés")

    # Titre principal
    tk.Label(root, text="Articles Crypto en temps réel", font=("Arial", 16, "bold")).pack(pady=10)

    # Filtrer par crypto
    crypto_filter = tk.StringVar(value="all")
    filter_frame = tk.Frame(root)
    tk.Label(filter_frame, text="Filtrer par crypto :", font=("Arial", 11)).pack(side="left", padx=(0, 5))
    tk.OptionMenu(filter_frame, crypto_filter, "all", "btc", "eth", "xrp", "ada", "sol", "doge", "bnb", "ltc").pack(side="left")
    filter_frame.pack(pady=5)

    # Zone de contenu défilable
    canvas = tk.Canvas(root, width=650, height=500)
    scrollbar = tk.Scrollbar(root, orient="vertical", command=canvas.yview)
    scrollable_frame = tk.Frame(canvas)

    scrollable_frame.bind("<Configure>",lambda e: canvas.configure(scrollregion=canvas.bbox("all")))

    canvas.create_window((0, 0), window=scrollable_frame, anchor="nw")
    canvas.configure(yscrollcommand=scrollbar.set)

    canvas.pack(side="left", fill="both", expand=True)
    scrollbar.pack(side="right", fill="y")

    # Barre de statut
    status_label = tk.Label(root, text="Chargement des articles...", font=("Arial", 10), fg="gray")
    status_label.pack(pady=(5, 0))

    # Bouton manuel pour recharger
    tk.Button(
        root,
        text="Rafraîchir maintenant",
        command=lambda: refresh_articles(scrollable_frame, canvas, status_label, crypto_filter)
    ).pack(pady=5)

    # Bouton quitter
    tk.Button(root, text="Quitter", command=root.destroy).pack(pady=10)

    # Premier chargement
    refresh_articles(scrollable_frame, canvas, status_label, crypto_filter)

    root.mainloop()


if __name__ == "__main__":
    main()
