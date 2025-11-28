import { http } from "./http";

export interface NewsArticle {
  time: string;
  source: string;
  url: string;
  title: string;
  content: string;
  sentiment_score: number | null;
  symbols: string[];
  created_at: string;
}

export class NewsService {
  static async getNewsBySymbol(
    symbol: string,
    limit = 100
  ): Promise<NewsArticle[]> {
    try {
      // Le backend Go attend un symbole encodé dans l'URL, ex: /api/v1/news/BTC%2FUSDT
      const encodedSymbol = encodeURIComponent(symbol);
      const response = await http.get(`/api/v1/news/${encodedSymbol}`, {
        params: { limit },
      });
      return response.data?.data || [];
    } catch (error) {
      console.error(`Error fetching news for symbol ${symbol}:`, error);
      return [];
    }
  }
}
