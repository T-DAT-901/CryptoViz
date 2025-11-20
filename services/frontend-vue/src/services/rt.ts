import type { CandleDTO } from "@/types/market";

const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8080/ws/crypto";
const USE_MOCK = import.meta.env.VITE_USE_MOCK === "true";

// Types WebSocket
export type WSMessageType =
  | "candle"
  | "trade"
  | "indicator"
  | "news"
  | "ack"
  | "error"
  | "pong"
  | "subscriptions";

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
    ["candle", "trade", "indicator", "news", "error", "ack"].forEach((type) => {
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
            const message: WSMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (err) {
            console.error("Failed to parse WebSocket message:", err);
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
