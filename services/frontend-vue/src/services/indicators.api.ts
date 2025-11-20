import { http } from "./http";
import {
  RSIPoint,
  MACDPoint,
  BollingerPoint,
  MomentumPoint,
} from "@/types/market";

const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true";

export async function fetchRSI(
  symbol: string,
  tf: string,
  period = 14
): Promise<RSIPoint[]> {
  if (USE_MOCK) {
    const rsiMock = await import("./mocks/ind_rsi.json");
    return rsiMock.default as RSIPoint[];
  }
  const { data }: { data: RSIPoint[] } = await http.get(
    `/api/v1/indicators/rsi`,
    {
      params: { symbol, tf, period },
    }
  );
  return data as RSIPoint[];
}

export async function fetchMACD(
  symbol: string,
  tf: string,
  fast = 12,
  slow = 26,
  signal = 9
): Promise<MACDPoint[]> {
  if (USE_MOCK) {
    const macdMock = await import("./mocks/ind_macd.json");
    return macdMock.default as MACDPoint[];
  }
  const { data }: { data: MACDPoint[] } = await http.get(
    `/api/v1/indicators/macd`,
    {
      params: { symbol, tf, fast, slow, signal },
    }
  );
  return data as MACDPoint[];
}

export async function fetchBollinger(
  symbol: string,
  tf: string,
  period = 20,
  stdDev = 2
): Promise<BollingerPoint[]> {
  if (USE_MOCK) {
    const bbMock = await import("./mocks/ind_bollinger.json");
    return bbMock.default as BollingerPoint[];
  }
  const { data }: { data: BollingerPoint[] } = await http.get(
    `/api/v1/indicators/bollinger`,
    { params: { symbol, tf, period, stdDev } }
  );
  return data as BollingerPoint[];
}

export async function fetchMomentum(
  symbol: string,
  tf: string,
  period = 10
): Promise<MomentumPoint[]> {
  if (USE_MOCK) {
    const momMock = await import("./mocks/ind_momentum.json");
    return momMock.default as MomentumPoint[];
  }
  const { data }: { data: MomentumPoint[] } = await http.get(
    `/api/v1/indicators/momentum`,
    { params: { symbol, tf, period } }
  );
  return data as MomentumPoint[];
}
