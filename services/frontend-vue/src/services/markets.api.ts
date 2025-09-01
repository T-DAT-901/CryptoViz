import { http } from "./http";
import candlesMock from "./mocks/candles.json";
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
    "/api/markets/tickers",
    { params: { symbols: symbols.join(",") } }
  );
  return data;
}

export async function fetchCandles(
  symbol: string,
  tf = "1m",
  limit = 120
): Promise<CandleDTO[]> {
  if (USE_MOCK) return candlesMock as CandleDTO[];
  const { data }: { data: CandleDTO[] } = await http.get(
    "/api/markets/candles",
    { params: { symbol, tf, limit } }
  );
  return data;
}

export async function fetchNews(): Promise<NewsDTO[]> {
  if (USE_MOCK) {
    return [] as NewsDTO[];
  }
  const { data }: { data: NewsDTO[] } = await http.get("/api/news");
  return data;
}
