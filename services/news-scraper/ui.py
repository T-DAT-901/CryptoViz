import tkinter as tk
from storage import load_articles
import webbrowser

REFRESH_INTERVAL = 60 * 1000  # 1 minute

def open_link(event, link):
    webbrowser.open(link)

def refresh_articles(content_frame, canvas):
    for widget in content_frame.winfo_children():
        widget.destroy()

    articles = load_articles().sort_values(by="published", ascending=False)

    for _, article in articles.iterrows():
        title_label = tk.Label(content_frame, text=article["title"], font=("Arial", 12, "bold"),
                               wraplength=600, justify="left", fg="black", cursor="hand2")
        title_label.pack(anchor="w", pady=(0,2))
        title_label.bind("<Button-1>", lambda e, l=article["link"]: open_link(e, l))

        info_label = tk.Label(content_frame, text=f"Publié le : {article['published']} | Sentiment: {article.get('sentiment', 'inconnu')}",
                              font=("Arial", 10), fg="gray")
        info_label.pack(anchor="w", pady=(0,10))

    content_frame.update_idletasks()
    canvas.config(scrollregion=canvas.bbox("all"))

    content_frame.after(REFRESH_INTERVAL, lambda: refresh_articles(content_frame, canvas))

def main():
    root = tk.Tk()
    root.title("Crypto News RSS - Articles sauvegardés")

    tk.Label(root, text="Articles Crypto en temps réel", font=("Arial", 16, "bold")).pack(pady=10)

    canvas = tk.Canvas(root, width=650, height=500)
    scrollbar = tk.Scrollbar(root, orient="vertical", command=canvas.yview)
    scrollable_frame = tk.Frame(canvas)

    scrollable_frame.bind(
        "<Configure>",
        lambda e: canvas.configure(scrollregion=canvas.bbox("all"))
    )

    window = canvas.create_window((0, 0), window=scrollable_frame, anchor="nw")
    canvas.configure(yscrollcommand=scrollbar.set)

    canvas.pack(side="left", fill="both", expand=True)
    scrollbar.pack(side="right", fill="y")

    refresh_articles(scrollable_frame, canvas)

    tk.Button(root, text="Quitter", command=root.destroy).pack(pady=10)

    root.mainloop()

if __name__ == "__main__":
    main()
