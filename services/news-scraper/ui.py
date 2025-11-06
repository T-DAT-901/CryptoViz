import tkinter as tk
from storage import load_articles
import webbrowser
from datetime import datetime

# Rafraîchissement automatique (en millisecondes)
REFRESH_INTERVAL = 60 * 1000  # 1 minute


def open_link(event, link: str):
    """Ouvre le lien de l'article dans le navigateur par défaut."""
    webbrowser.open(link)


def refresh_articles(content_frame, canvas, status_label=None):
    """Rafraîchit la liste des articles dans l'interface."""
    # Affiche un message temporaire pendant le rafraîchissement
    if status_label:
        status_label.config(text="🔄 Actualisation en cours...")
        status_label.update_idletasks()

    # Vide la frame
    for widget in content_frame.winfo_children():
        widget.destroy()

    # Récupère et trie les articles du plus récent au plus ancien
    articles = sorted(load_articles(), key=lambda x: x["published"], reverse=True)

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

        info_text = f"Publié le : {article['published']}"
        if "sentiment" in article:
            info_text += f" | Sentiment : {article['sentiment']}"

        info_label = tk.Label(
            content_frame,
            text=info_text,
            font=("Arial", 10),
            fg="gray"
        )
        info_label.pack(anchor="w", pady=(0, 10))

    # Met à jour la zone de scroll
    content_frame.update_idletasks()
    canvas.config(scrollregion=canvas.bbox("all"))

    # Met à jour le statut
    if status_label:
        now = datetime.now().strftime("%H:%M:%S")
        status_label.config(text=f"✅ Dernière mise à jour : {now}")

    # Programme la prochaine actualisation
    content_frame.after(REFRESH_INTERVAL, lambda: refresh_articles(content_frame, canvas, status_label))


def main():
    """Lance l'interface principale."""
    root = tk.Tk()
    root.title("🪙 Crypto News RSS - Articles sauvegardés")

    # En-tête
    tk.Label(root, text="Articles Crypto en temps réel", font=("Arial", 16, "bold")).pack(pady=10)

    # Zone de contenu défilable
    canvas = tk.Canvas(root, width=650, height=500)
    scrollbar = tk.Scrollbar(root, orient="vertical", command=canvas.yview)
    scrollable_frame = tk.Frame(canvas)

    scrollable_frame.bind(
        "<Configure>",
        lambda e: canvas.configure(scrollregion=canvas.bbox("all"))
    )

    canvas.create_window((0, 0), window=scrollable_frame, anchor="nw")
    canvas.configure(yscrollcommand=scrollbar.set)

    canvas.pack(side="left", fill="both", expand=True)
    scrollbar.pack(side="right", fill="y")

    # Barre de statut
    status_label = tk.Label(root, text="Chargement des articles...", font=("Arial", 10), fg="gray")
    status_label.pack(pady=(5, 0))

    # Rafraîchit la liste au lancement
    refresh_articles(scrollable_frame, canvas, status_label)

    # Bouton quitter
    tk.Button(root, text="Quitter", command=root.destroy).pack(pady=10)

    root.mainloop()


if __name__ == "__main__":
    main()
