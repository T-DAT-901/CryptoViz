import { defineStore } from "pinia";
import type { TickerDTO, CandleDTO } from "@/types/market";

export const useMarketStore = defineStore("market", {
  state: () => ({
    interval: "1m" as string,
    candles: {} as Record<string, CandleDTO[]>,
    tickers: {} as Record<string, TickerDTO>,
  }),

  actions: {
    setInterval(interval: string) {
      this.interval = interval;
    },
    upsertTicker(ticker: TickerDTO) {
      this.tickers[ticker.symbol] = ticker;
    },
    setCandles(symbol: string, candles: CandleDTO[]) {
      this.candles[symbol] = candles;
    },
    addCandles(symbol: string, candle: CandleDTO) {
      if (!this.candles[symbol]) {
        this.candles[symbol] = [];
      }
      this.candles[symbol].push(candle);
    },
  },
});
