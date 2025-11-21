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
    // Backend-go uses query parameters: /api/v1/crypto/data?symbol=BTC/USDT&interval=1m&limit=120
    const response = await http.get(`/api/v1/crypto/data`, {
      params: { symbol, interval, limit },
    });
    // Transform backend format to chart format
    const backendData = response.data?.data || [];
    return backendData.map((candle: any) => ({
      time: candle.window_start || candle.time,
      open: candle.open,
      high: candle.high,
      low: candle.low,
      close: candle.close,
      volume: candle.volume,
    }));
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
    // Backend-go uses query parameters: /api/v1/indicators/{type}?symbol=BTC/USDT
    const response = await http.get(`/api/v1/indicators/${type}`, {
      params: { symbol },
    });
    return response.data?.data || [];
  } catch (error) {
    console.error(`Error fetching ${type} for ${symbol}:`, error);
    return [];
  }
}
