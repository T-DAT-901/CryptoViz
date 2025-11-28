import { ref } from "vue";
import type { CandleDTO } from "@/types/market";

// Types WebSocket
interface WebSocketMessage {
  type: "price_update" | "candle_update" | "heartbeat";
  data: any;
  symbol: string;
  timeframe: string;
  timestamp: number;
}

interface PriceUpdate {
  symbol: string;
  price: number;
  change: number;
  changePercent: number;
  volume: number;
  high24h: number;
  low24h: number;
}

// État global WebSocket
const isConnected = ref(false);
const lastUpdate = ref<Date | null>(null);
const subscribers = new Map<string, Set<(data: any) => void>>();

class TradingWebSocket {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private heartbeatInterval: NodeJS.Timeout | null = null;

  constructor(private baseUrl = "") {
    this.baseUrl =
      baseUrl || import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/crypto";
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.baseUrl);

        this.ws.onopen = () => {
          console.log("📡 WebSocket connecté");
          isConnected.value = true;
          this.reconnectAttempts = 0;
          this.startHeartbeat();
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
            lastUpdate.value = new Date();
          } catch (error) {
            console.error("Erreur parsing WebSocket:", error);
          }
        };

        this.ws.onclose = () => {
          console.log("📡 WebSocket fermé");
          isConnected.value = false;
          this.stopHeartbeat();
          this.handleReconnect();
        };

        this.ws.onerror = (error) => {
          console.error("Erreur WebSocket:", error);
          reject(error);
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  private handleMessage(message: any) {
    if (message.type === "price_update") {
      this.notifySubscribers("price_update", message);
    } else if (message.type === "candle_update") {
      this.notifySubscribers("candle_update", message.data);
    }
  }

  private notifySubscribers(type: string, data: any) {
    const typeSubscribers = subscribers.get(type);
    if (typeSubscribers) {
      typeSubscribers.forEach((callback) => {
        try {
          callback(data);
        } catch (error) {
          console.error("Erreur callback subscriber:", error);
        }
      });
    }
  }

  private startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        // Binance ne nécessite pas de ping manuel
        // this.ws.send(JSON.stringify({ method: "ping" }));
      }
    }, 30000); // 30s
  }

  private stopHeartbeat() {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private handleReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay =
        this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

      console.log(
        `🔄 Tentative de reconnexion ${this.reconnectAttempts}/${this.maxReconnectAttempts} dans ${delay}ms`
      );

      setTimeout(() => {
        this.connect().catch(() => {});
      }, delay);
    } else {
      console.error(
        "❌ Échec de reconnexion WebSocket après",
        this.maxReconnectAttempts,
        "tentatives"
      );
    }
  }

  subscribe(type: string, callback: (data: any) => void): () => void {
    if (!subscribers.has(type)) {
      subscribers.set(type, new Set());
    }

    subscribers.get(type)!.add(callback);

    return () => {
      const typeSubscribers = subscribers.get(type);
      if (typeSubscribers) {
        typeSubscribers.delete(callback);
        if (typeSubscribers.size === 0) {
          subscribers.delete(type);
        }
      }
    };
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.stopHeartbeat();
    subscribers.clear();
  }

  getConnectionStatus() {
    return {
      isConnected: isConnected.value,
      lastUpdate: lastUpdate.value,
      subscriberCount: Array.from(subscribers.values()).reduce(
        (total, set) => total + set.size,
        0
      ),
    };
  }
}

// Instance singleton
const tradingWS = new TradingWebSocket();

// Hooks composables pour Vue
export function useTradingWebSocket() {
  return {
    connect: () => tradingWS.connect(),
    disconnect: () => tradingWS.disconnect(),
    subscribe: (type: string, callback: (data: any) => void) =>
      tradingWS.subscribe(type, callback),
    isConnected: isConnected,
    lastUpdate: lastUpdate,
    getStatus: () => tradingWS.getConnectionStatus(),
  };
}

// Hook pour les prix en temps réel
export function useLivePrices(symbol = "BTCUSDT") {
  const priceData = ref<PriceUpdate | null>(null);

  const { subscribe } = useTradingWebSocket();

  const unsubscribe = subscribe("price_update", (update: PriceUpdate) => {
    if (update.symbol === symbol) {
      priceData.value = update;
    }
  });

  return {
    priceData,
    unsubscribe,
  };
}

// Hook pour les bougies en temps réel
export function useLiveCandles() {
  const latestCandle = ref<CandleDTO | null>(null);

  const { subscribe } = useTradingWebSocket();

  const unsubscribe = subscribe("candle_update", (candle: CandleDTO) => {
    latestCandle.value = candle;
  });

  return {
    latestCandle,
    unsubscribe,
  };
}

// Hook pour les trades en temps réel (pour peupler le graph)
export function useLiveTrades(symbol = "BTCUSDT") {
  const trades = ref<Array<{ timestamp: number; price: number }>>([]);

  const { subscribe } = useTradingWebSocket();

  const unsubscribe = subscribe("trade", (trade: any) => {
    // trade devrait avoir: { symbol, price, timestamp, ... }
    if (trade.symbol === symbol) {
      trades.value.push({
        timestamp: new Date(trade.timestamp).getTime(),
        price: parseFloat(trade.price) || 0,
      });

      // Garder seulement les 1000 derniers trades
      if (trades.value.length > 1000) {
        trades.value = trades.value.slice(-1000);
      }
    }
  });

  return {
    trades,
    unsubscribe,
  };
}

export default tradingWS;
