import type { CandleDTO } from "@/types/market";

const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/crypto";
const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true"; // true seulement si "true"

console.log(
  "🔧 RT Config - WS_URL:",
  WS_URL,
  "USE_MOCK:",
  USE_MOCK,
  "VITE_USE_MOCK env:",
  import.meta.env.VITE_USE_MOCK
);

// Types WebSocket
export type WSMessageType =
  | "candle"
  | "trade"
  | "indicator"
  | "news"
  | "stats"
  | "ack"
  | "error"
  | "pong"
  | "subscriptions"
  | "crypto_list";

export interface WSMessage {
  type: WSMessageType;
  data: unknown;
  timestamp: number;
}

export interface ClientMessage {
  action: "subscribe" | "unsubscribe" | "ping" | "list_subscriptions";
  type?: string;
  symbol?: string;
  timeframe?: string;
}

// Real-time Data Service
export class RTClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;
  private messageHandlers: Map<string, Set<(data: WSMessage) => void>> =
    new Map();
  private isConnecting = false;
  private subscriptions = new Set<string>();

  constructor() {
    this.initializeHandlers();
  }

  private initializeHandlers() {
    // Initialize handler sets for each message type
    [
      "candle",
      "trade",
      "indicator",
      "news",
      "stats",
      "error",
      "ack",
      "crypto_list",
    ].forEach((type) => {
      this.messageHandlers.set(type, new Set());
    });
  }

  // Connect to WebSocket
  connect(): Promise<void> {
    if (USE_MOCK) {
      console.warn("Real-time disabled in mock mode");
      return Promise.resolve();
    }

    return new Promise((resolve, reject) => {
      if (this.isConnecting) {
        reject(new Error("Already connecting"));
        return;
      }

      this.isConnecting = true;

      try {
        this.ws = new WebSocket(WS_URL);

        this.ws.onopen = () => {
          console.log("✅ WebSocket connected");
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            // Debug: Log raw message
            console.log("📨 Raw WS message:", event.data.substring(0, 200));

            // Try to parse as single JSON first
            try {
              const message: WSMessage = JSON.parse(event.data);
              this.handleMessage(message);
            } catch (singleError) {
              // If single parse fails, try to extract multiple JSON objects
              const data = event.data;
              let startIdx = 0;
              let braceCount = 0;
              let inString = false;
              let escapeNext = false;

              for (let i = 0; i < data.length; i++) {
                const char = data[i];

                if (escapeNext) {
                  escapeNext = false;
                  continue;
                }

                if (char === "\\" && inString) {
                  escapeNext = true;
                  continue;
                }

                if (char === '"' && !escapeNext) {
                  inString = !inString;
                  continue;
                }

                if (!inString) {
                  if (char === "{") {
                    if (braceCount === 0) startIdx = i;
                    braceCount++;
                  } else if (char === "}") {
                    braceCount--;
                    if (braceCount === 0) {
                      try {
                        const jsonStr = data.substring(startIdx, i + 1);
                        const message: WSMessage = JSON.parse(jsonStr);
                        this.handleMessage(message);
                      } catch (parseError) {
                        console.warn(
                          "⚠️ Could not parse JSON chunk:",
                          data.substring(startIdx, i + 1)
                        );
                      }
                    }
                  }
                }
              }
            }
          } catch (err) {
            console.error("❌ Failed to process WebSocket message:", err);
            console.error("Raw data:", event.data.substring(0, 500));
          }
        };

        this.ws.onerror = (error) => {
          console.error("❌ WebSocket error:", error);
          this.isConnecting = false;
          reject(error);
        };

        this.ws.onclose = () => {
          console.log("WebSocket disconnected");
          this.ws = null;
          this.isConnecting = false;
          this.attemptReconnect();
        };
      } catch (err) {
        this.isConnecting = false;
        reject(err);
      }
    });
  }

  // Reconnect logic
  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("Max reconnect attempts reached");
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    console.log(
      `Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    );

    setTimeout(() => {
      this.connect().catch(console.error);
    }, delay);
  }

  // Subscribe to data
  subscribe(type: string, symbol: string, timeframe = ""): void {
    const msg: ClientMessage = {
      action: "subscribe",
      type,
      symbol,
      timeframe,
    };

    const key = `${type}:${symbol}:${timeframe}`;
    this.subscriptions.add(key);
    this.sendMessage(msg);
  }

  // Unsubscribe from data
  unsubscribe(type: string, symbol: string, timeframe = ""): void {
    const msg: ClientMessage = {
      action: "unsubscribe",
      type,
      symbol,
      timeframe,
    };

    const key = `${type}:${symbol}:${timeframe}`;
    this.subscriptions.delete(key);
    this.sendMessage(msg);
  }

  // List subscriptions
  listSubscriptions(): void {
    this.sendMessage({ action: "list_subscriptions" });
  }

  // Send ping
  ping(): void {
    this.sendMessage({ action: "ping" });
  }

  // Send message to server
  private sendMessage(msg: ClientMessage): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn("WebSocket not connected");
      return;
    }

    try {
      this.ws.send(JSON.stringify(msg));
    } catch (err) {
      console.error("Failed to send message:", err);
    }
  }

  // Handle incoming messages
  private handleMessage(msg: WSMessage) {
    const handlers = this.messageHandlers.get(msg.type);
    if (handlers) {
      handlers.forEach((handler) => handler(msg));
    }
  }

  // Register message handler
  on(type: WSMessageType, handler: (data: WSMessage) => void): () => void {
    const handlers = this.messageHandlers.get(type);
    if (handlers) {
      handlers.add(handler);

      // Return unsubscribe function
      return () => {
        handlers.delete(handler);
      };
    }
    return () => {};
  }

  // Disconnect
  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.subscriptions.clear();
  }

  // Get connection status
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  // Get current subscriptions
  getSubscriptions(): string[] {
    return Array.from(this.subscriptions);
  }

  // Request list of all cryptocurrencies
  async requestCryptoList(): Promise<any[]> {
    return new Promise((resolve, reject) => {
      if (!this.isConnected()) {
        reject(new Error("WebSocket not connected"));
        return;
      }

      // Register a one-time handler for the response
      const handler = (message: WSMessage) => {
        if (message.type === "crypto_list") {
          // Remove the handler after receiving the response
          const handlers = this.messageHandlers.get("crypto_list");
          if (handlers) {
            handlers.delete(handler);
          }
          resolve(message.data as any[]);
        }
      };

      this.on("crypto_list", handler);

      // Send the request
      const request: ClientMessage = {
        action: "list_subscriptions", // Or a custom action like "get_crypto_list"
        type: "crypto_list",
      };

      this.ws?.send(JSON.stringify(request));

      // Timeout after 5 seconds
      setTimeout(() => {
        const handlers = this.messageHandlers.get("crypto_list");
        if (handlers) {
          handlers.delete(handler);
        }
        reject(new Error("WebSocket request timeout"));
      }, 5000);
    });
  }
}

// Singleton instance
let rtClient: RTClient | null = null;

export function getRTClient(): RTClient {
  if (!rtClient) {
    rtClient = new RTClient();
  }
  return rtClient;
}

// Utility function to disconnect
export function disconnectRT(): void {
  rtClient?.disconnect();
  rtClient = null;
}

// Real-time Candle Builder for 1m timeframe
export class RealtimeCandleBuilder {
  private currentMinute: number = 0;
  private candle = {
    open: 0,
    high: 0,
    low: Infinity,
    close: 0,
    volume: 0,
  };
  private trades: Array<{ price: number; volume: number; timestamp: number }> =
    [];
  private onCandleComplete:
    | ((candle: typeof this.candle, timestamp: number) => void)
    | null = null;
  private onCandleUpdate: ((candle: typeof this.candle) => void) | null = null;

  constructor() {
    this.initializeNewMinute();
  }

  private initializeNewMinute() {
    const now = new Date();
    this.currentMinute = Math.floor(now.getTime() / 60000) * 60000;
    this.candle = {
      open: 0,
      high: 0,
      low: Infinity,
      close: 0,
      volume: 0,
    };
    this.trades = [];
  }

  addTrade(price: number, volume: number, timestamp: number) {
    const tradeMinute = Math.floor(timestamp / 60000) * 60000;

    // Si le trade est d'une minute différente, finaliser la candle actuelle
    if (tradeMinute > this.currentMinute) {
      if (this.trades.length > 0 && this.onCandleComplete) {
        this.finalizCandle();
      }
      this.initializeNewMinute();
    }

    // Si c'est le premier trade de la minute, set l'open
    if (this.trades.length === 0) {
      this.candle.open = price;
    }

    // Update OHLCV
    this.candle.high = Math.max(this.candle.high, price);
    this.candle.low = Math.min(this.candle.low, price);
    this.candle.close = price;
    this.candle.volume += volume;

    this.trades.push({ price, volume, timestamp });

    // Notifier les updates
    if (this.onCandleUpdate) {
      this.onCandleUpdate({ ...this.candle });
    }
  }

  private finalizCandle() {
    if (this.onCandleComplete) {
      this.onCandleComplete(
        {
          ...this.candle,
          low:
            this.candle.low === Infinity ? this.candle.open : this.candle.low,
        },
        this.currentMinute
      );
    }
  }

  getCurrentCandle() {
    return {
      ...this.candle,
      low: this.candle.low === Infinity ? this.candle.open : this.candle.low,
    };
  }

  onCandleCompleted(
    callback: (candle: typeof this.candle, timestamp: number) => void
  ) {
    this.onCandleComplete = callback;
  }

  onCandleUpdated(callback: (candle: typeof this.candle) => void) {
    this.onCandleUpdate = callback;
  }

  reset() {
    this.initializeNewMinute();
  }
}
