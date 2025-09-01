# ui.py
import tkinter as tk
from storage import load_articles
import webbrowser

REFRESH_INTERVAL = 60 * 1000  # 1 minute en millisecondes

def open_link(event, link):
    webbrowser.open(link)

def refresh_articles(frame):
    # Supprime tout ce qui est affiché
    for widget in frame.winfo_children():
        widget.destroy()

    articles = load_articles().sort_values(by="published", ascending=False)

    for _, article in articles.iterrows():
        title_label = tk.Label(frame, text=article["title"], font=("Arial", 12, "bold"), wraplength=600, justify="left", fg="black", cursor="hand2")
        title_label.pack(anchor="w")
        title_label.bind("<Button-1>", lambda e, link=article["link"]: open_link(e, link))

        info_label = tk.Label(frame, text=f"Publié le : {article['published']}", font=("Arial", 10), fg="gray")
        info_label.pack(anchor="w", pady=(0,10))

    # Relance la mise à jour après X ms
    frame.after(REFRESH_INTERVAL, lambda: refresh_articles(frame))

def main():
    root = tk.Tk()
    root.title("Crypto News RSS - Temps Réel")

    tk.Label(root, text="Articles Crypto en Temps Réel", font=("Arial", 16, "bold")).pack(pady=10)

    frame = tk.Frame(root)
    frame.pack(padx=10, pady=10)

    refresh_articles(frame)  # première mise à jour

    tk.Button(root, text="Quitter", command=root.destroy).pack(pady=10)

    root.mainloop()

if __name__ == "__main__":
    main()
