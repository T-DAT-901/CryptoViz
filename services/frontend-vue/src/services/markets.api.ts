import { http } from "./http";
import type { CandleDTO, TickerDTO, NewsDTO } from "@/types/market";

const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true";

export async function fetchTickers(symbols: string[]): Promise<TickerDTO[]> {
  if (USE_MOCK) {
    const tickers = symbols.map((s) => ({
      symbol: s,
      price: 0,
      ts: Date.now(),
    }));
    return tickers;
  }
  const { data }: { data: TickerDTO[] } = await http.get(
    "/api/v1/markets/tickers",
    { params: { symbols: symbols.join(",") } }
  );
  return data;
}

export async function fetchCandles(
  symbol: string,
  interval = "1m",
  limit = 120
): Promise<CandleDTO[]> {
  if (USE_MOCK) {
    const candlesMock = await import("./mocks/candles.json");
    return candlesMock.default as CandleDTO[];
  }
  try {
    // Encoder le symbole pour l'URL (BTC/FDUSD -> BTC%2FFDUSD)
    const encodedSymbol = encodeURIComponent(symbol);
    const response = await http.get(`/api/v1/crypto/${encodedSymbol}/data`, {
      params: { interval, limit },
    });
    // La réponse du backend est { success: true, data: { symbol, interval, data: [] } }
    return response.data?.data?.data || [];
  } catch (error) {
    console.error(`Error fetching candles for ${symbol}:`, error);
    return [];
  }
}

export async function fetchNews(): Promise<NewsDTO[]> {
  if (USE_MOCK) {
    const newsMock = await import("./mocks/news.json");
    return (newsMock.default as NewsDTO[]).slice(0, 15);
  }
  const response = await http.get("/api/v1/news");
  return response.data.data || [];
}

// Charger les indicateurs depuis l'API
export async function fetchIndicators(
  symbol: string,
  type: "rsi" | "macd" | "bollinger" | "momentum"
): Promise<any[]> {
  if (USE_MOCK) {
    const mockName = `ind_${type}`;
    const indicatorMock = await import(`./mocks/${mockName}.json`);
    return indicatorMock.default as any[];
  }
  try {
    const encodedSymbol = encodeURIComponent(symbol);
    const response = await http.get(
      `/api/v1/indicators/${encodedSymbol}/${type}`
    );
    return response.data?.data || [];
  } catch (error) {
    console.error(`Error fetching ${type} for ${symbol}:`, error);
    return [];
  }
}
