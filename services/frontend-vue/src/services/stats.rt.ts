import type { WSMessage } from "./rt";

const STATS_WS_URL =
  import.meta.env.VITE_WS_URL?.replace("/ws/crypto", "/ws/stats") ||
  "ws://localhost:8080/ws/stats";
const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true";

console.log(
  "🔧 Stats RT Config - WS_URL:",
  STATS_WS_URL,
  "USE_MOCK:",
  USE_MOCK
);

export interface StatsData {
  symbol: string;
  change1h: number;
  change24h: number;
  change7d: number;
  marketCap: number;
  volume24h: number;
  circulatingSupply: number;
}

// Real-time Stats Service
export class StatsRTClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 3000;
  private messageHandlers: Map<string, Set<(data: StatsData) => void>> =
    new Map();
  private isConnecting = false;
  private subscriptions = new Set<string>();

  constructor() {
    this.initializeHandlers();
  }

  private initializeHandlers() {
    // Initialize handler set for stats
    this.messageHandlers.set("stats", new Set());
  }

  // Connect to WebSocket
  connect(): Promise<void> {
    if (USE_MOCK) {
      console.warn("Real-time stats disabled in mock mode");
      return Promise.resolve();
    }

    return new Promise((resolve, reject) => {
      if (this.isConnecting) {
        reject(new Error("Already connecting"));
        return;
      }

      this.isConnecting = true;

      try {
        this.ws = new WebSocket(STATS_WS_URL);

        this.ws.onopen = () => {
          console.log("✅ Stats WebSocket connected");
          this.isConnecting = false;
          this.reconnectAttempts = 0;
          resolve();
        };

        this.ws.onmessage = (event) => {
          try {
            console.log(
              "📊 Raw Stats WS message:",
              event.data.substring(0, 150)
            );

            // Try to parse as single JSON first
            try {
              const message: any = JSON.parse(event.data);
              if (message.data && message.data.symbol) {
                this.handleMessage(message.data as StatsData);
              }
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
                        const message: any = JSON.parse(jsonStr);
                        if (message.data && message.data.symbol) {
                          this.handleMessage(message.data as StatsData);
                        }
                      } catch (parseError) {
                        console.warn(
                          "⚠️ Could not parse stats JSON chunk:",
                          data.substring(
                            startIdx,
                            Math.min(startIdx + 100, i + 1)
                          )
                        );
                      }
                    }
                  }
                }
              }
            }
          } catch (err) {
            console.error("❌ Failed to process stats message:", err);
          }
        };

        this.ws.onerror = (error) => {
          console.error("❌ Stats WebSocket error:", error);
          this.isConnecting = false;
          reject(error);
        };

        this.ws.onclose = () => {
          console.log("WebSocket stats disconnected");
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
      console.error("Max stats reconnect attempts reached");
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    console.log(
      `Reconnecting stats in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`
    );

    setTimeout(() => {
      this.connect().catch(console.error);
    }, delay);
  }

  // Subscribe to stats
  subscribe(symbol: string): void {
    const msg = {
      action: "subscribe",
      type: "stats",
      symbol,
    };

    const key = `stats:${symbol}`;
    this.subscriptions.add(key);
    this.sendMessage(msg);
  }

  // Unsubscribe from stats
  unsubscribe(symbol: string): void {
    const msg = {
      action: "unsubscribe",
      type: "stats",
      symbol,
    };

    const key = `stats:${symbol}`;
    this.subscriptions.delete(key);
    this.sendMessage(msg);
  }

  // Send message to server
  private sendMessage(msg: any): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn("Stats WebSocket not connected");
      return;
    }

    try {
      this.ws.send(JSON.stringify(msg));
    } catch (err) {
      console.error("Failed to send stats message:", err);
    }
  }

  // Handle incoming messages
  private handleMessage(data: StatsData) {
    const handlers = this.messageHandlers.get("stats");
    if (handlers) {
      handlers.forEach((handler) => handler(data));
    }
  }

  // Register message handler
  on(handler: (data: StatsData) => void): () => void {
    const handlers = this.messageHandlers.get("stats");
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
}

// Singleton instance
let statsRTClient: StatsRTClient | null = null;

export function getStatsRTClient(): StatsRTClient {
  if (!statsRTClient) {
    statsRTClient = new StatsRTClient();
  }
  return statsRTClient;
}

// Utility function to disconnect
export function disconnectStatsRT(): void {
  statsRTClient?.disconnect();
  statsRTClient = null;
}
