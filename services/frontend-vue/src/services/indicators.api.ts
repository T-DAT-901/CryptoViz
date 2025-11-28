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
  period = 14,
  limit = 10000,
  startTime?: string,
  endTime?: string
): Promise<RSIPoint[]> {
  if (USE_MOCK) {
    const rsiMock = await import("./mocks/ind_rsi.json");
    return rsiMock.default as RSIPoint[];
  }
  // Backend-go uses query parameters: /api/v1/indicators/rsi?symbol=BTC/USDT&interval=1m&limit=1000
  // Optional time range: start_time and end_time in RFC3339 format
  const params: any = { symbol, interval: tf, limit };
  if (startTime) params.start_time = startTime;
  if (endTime) params.end_time = endTime;

  const response = await http.get(`/api/v1/indicators/rsi`, { params });
  return response.data?.data || [];
}

export async function fetchMACD(
  symbol: string,
  tf: string,
  fast = 12,
  slow = 26,
  signal = 9,
  limit = 10000,
  startTime?: string,
  endTime?: string
): Promise<MACDPoint[]> {
  if (USE_MOCK) {
    const macdMock = await import("./mocks/ind_macd.json");
    return macdMock.default as MACDPoint[];
  }
  // Backend-go uses query parameters: /api/v1/indicators/macd?symbol=BTC/USDT&interval=1m&limit=1000
  // Optional time range: start_time and end_time in RFC3339 format
  const params: any = { symbol, interval: tf, limit };
  if (startTime) params.start_time = startTime;
  if (endTime) params.end_time = endTime;

  const response = await http.get(`/api/v1/indicators/macd`, { params });
  return response.data?.data || [];
}

export async function fetchBollinger(
  symbol: string,
  tf: string,
  period = 20,
  stdDev = 2,
  limit = 10000,
  startTime?: string,
  endTime?: string
): Promise<BollingerPoint[]> {
  if (USE_MOCK) {
    const bbMock = await import("./mocks/ind_bollinger.json");
    return bbMock.default as BollingerPoint[];
  }
  // Backend-go uses query parameters: /api/v1/indicators/bollinger?symbol=BTC/USDT&interval=1m&limit=1000
  // Optional time range: start_time and end_time in RFC3339 format
  const params: any = { symbol, interval: tf, limit };
  if (startTime) params.start_time = startTime;
  if (endTime) params.end_time = endTime;

  const response = await http.get(`/api/v1/indicators/bollinger`, { params });
  return response.data?.data || [];
}

export async function fetchMomentum(
  symbol: string,
  tf: string,
  period = 10,
  limit = 10000,
  startTime?: string,
  endTime?: string
): Promise<MomentumPoint[]> {
  if (USE_MOCK) {
    const momMock = await import("./mocks/ind_momentum.json");
    return momMock.default as MomentumPoint[];
  }
  // Backend-go uses query parameters: /api/v1/indicators/momentum?symbol=BTC/USDT&interval=1m&limit=1000
  // Optional time range: start_time and end_time in RFC3339 format
  const params: any = { symbol, interval: tf, limit };
  if (startTime) params.start_time = startTime;
  if (endTime) params.end_time = endTime;

  const response = await http.get(`/api/v1/indicators/momentum`, { params });
  return response.data?.data || [];
}
