import { defineStore } from "pinia";
import { getRTClient, disconnectRT } from "@/services/rt";
import type { TickerDTO, CandleDTO } from "@/types/market";
import type { WSMessage } from "@/services/rt";

export const useMarketStore = defineStore("market", {
  state: () => ({
    interval: "1m" as string,
    candles: {} as Record<string, CandleDTO[]>,
    tickers: {} as Record<string, TickerDTO>,
    rtConnected: false,
    activeSymbol: "BTC/USDT" as string, // Backend-go uses BASE/QUOTE format
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

      // Limit history to last 500 candles for performance
      const candles = this.candles[symbol];
      candles.push(candle);
      if (candles.length > 500) {
        candles.shift();
      }
    },

    // WebSocket actions
    async connectRealtime() {
      try {
        const rt = getRTClient();
        await rt.connect();
        this.rtConnected = true;

        // Subscribe to price updates (backend-go uses "trade" and "candle")
        rt.subscribe("trade", this.activeSymbol);
        rt.subscribe("candle", this.activeSymbol, this.interval);

        // Listen to candles
        rt.on("candle", (msg: WSMessage) => {
          const rawCandle = msg.data as any;
          // Transform backend format to frontend format
          const candle: CandleDTO = {
            time: rawCandle.window_start || rawCandle.time,
            open: rawCandle.open,
            high: rawCandle.high,
            low: rawCandle.low,
            close: rawCandle.close,
            volume: rawCandle.volume,
          };
          this.addCandles(this.activeSymbol, candle);
        });

        // Listen to trades (for latest price)
        rt.on("trade", (msg: WSMessage) => {
          const trade = msg.data as any;
          if (trade.symbol === this.activeSymbol) {
            this.upsertTicker({
              symbol: trade.symbol,
              price: trade.price,
              ts: Date.now(),
            });
          }
        });

        // Listen to errors
        rt.on("error", (msg: WSMessage) => {
          console.error("WebSocket error:", msg.data);
        });
      } catch (err) {
        console.error("Failed to connect to realtime:", err);
        this.rtConnected = false;
      }
    },

    disconnectRealtime() {
      disconnectRT();
      this.rtConnected = false;
    },

    switchSymbol(symbol: string) {
      const rt = getRTClient();

      // Unsubscribe from old symbol
      if (this.activeSymbol !== symbol) {
        rt.unsubscribe("trade", this.activeSymbol);
        rt.unsubscribe("candle", this.activeSymbol, this.interval);

        // Subscribe to new symbol
        rt.subscribe("trade", symbol);
        rt.subscribe("candle", symbol, this.interval);

        this.activeSymbol = symbol;
        this.candles[symbol] = this.candles[symbol] || [];
      }
    },

    switchInterval(interval: string) {
      const rt = getRTClient();

      // Unsubscribe from old interval
      if (this.interval !== interval) {
        rt.unsubscribe("candle", this.activeSymbol, this.interval);

        // Subscribe to new interval
        rt.subscribe("candle", this.activeSymbol, interval);

        this.interval = interval;
      }
    },
  },
});
