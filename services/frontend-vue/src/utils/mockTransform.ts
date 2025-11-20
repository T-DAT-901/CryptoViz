import type { CandleDTO } from "@/types/market";

// Transform old mock format (t, o, h, l, c) to new CandleDTO format (time, open, high, low, close, volume)
export function transformOldCandleFormat(oldCandle: any): CandleDTO {
  return {
    time: new Date(oldCandle.t).toISOString(),
    open: oldCandle.o,
    high: oldCandle.h,
    low: oldCandle.l,
    close: oldCandle.c,
    volume: oldCandle.v || 0,
  };
}

export function transformOldCandlesArray(oldCandles: any[]): CandleDTO[] {
  return oldCandles.map(transformOldCandleFormat);
}
