import { defineStore } from "pinia";
import { getRTClient } from "@/services/rt";
import type { CryptoData } from "@/services/crypto.api";

export const useCryptoStore = defineStore("crypto", {
  state: () => ({
    cryptos: [] as CryptoData[],
    cryptoMap: new Map<string, CryptoData>(),
    loading: false,
    rtConnected: false,
  }),

  getters: {
    sortedCryptos(): CryptoData[] {
      return Array.from(this.cryptoMap.values()).sort(
        (a, b) => a.rank - b.rank
      );
    },
  },

  actions: {
    /**
     * Connecte au WebSocket et commence à écouter les trades
     */
    async initializeFromWebSocket() {
      try {
        this.loading = true;
        const rt = getRTClient();

        // Connecte si pas encore connecté
        if (!rt.isConnected()) {
          console.log("📡 Connecting to WebSocket for crypto data...");
          await rt.connect();
        }

        this.rtConnected = true;

        console.log("📡 Subscribing to all trades...");
        rt.subscribe("trade", "*");

        // Handler pour les trades
        const onTrade = (message: any) => {
          if (message.data && message.data.symbol && message.data.price) {
            const symbol = message.data.symbol;
            const price = message.data.price;

            console.log("✨ Trade received:", symbol, "@", price);

            // Récupère ou crée l'entrée crypto
            if (!this.cryptoMap.has(symbol)) {
              const newCrypto: CryptoData = {
                id: symbol,
                rank: this.cryptoMap.size + 1,
                name: symbol.split("/")[0], // BTC de BTC/USDT
                symbol: symbol,
                price: price,
                change1h: 0,
                change24h: 0,
                change7d: 0,
                marketCap: 0,
                volume24h: 0,
                circulatingSupply: 0,
                sparklineData: [],
              };
              this.cryptoMap.set(symbol, newCrypto);
              console.log("➕ Added new crypto:", symbol);
            } else {
              // Met à jour le prix
              const existing = this.cryptoMap.get(symbol)!;
              existing.price = price;
              this.cryptoMap.set(symbol, existing);
            }
          }
        };

        // S'enregistrer au handler des trades
        rt.on("trade", onTrade);

        // Attendre un peu pour laisser les trades arriver
        await new Promise((resolve) => setTimeout(resolve, 5000));

        console.log("✅ Loaded", this.cryptoMap.size, "cryptos from WebSocket");
        this.loading = false;
      } catch (error) {
        console.error("❌ Error initializing crypto store:", error);
        this.loading = false;

        // Fallback sur les mocks
        try {
          const cryptosData = await import("@/services/mocks/cryptos.json");
          this.cryptoMap.clear();
          cryptosData.default.forEach((crypto: CryptoData) => {
            this.cryptoMap.set(crypto.symbol, crypto);
          });
          console.log("✅ Loaded mock cryptos:", this.cryptoMap.size, "items");
        } catch (mockError) {
          console.error("❌ Error loading mock cryptos:", mockError);
        }
      }
    },

    /**
     * Ajoute ou met à jour une crypto
     */
    upsertCrypto(crypto: CryptoData) {
      this.cryptoMap.set(crypto.symbol, crypto);
    },

    /**
     * Retourne une crypto par symbole
     */
    getCrypto(symbol: string): CryptoData | undefined {
      return this.cryptoMap.get(symbol);
    },

    /**
     * Nettoie les données
     */
    clear() {
      this.cryptoMap.clear();
      this.cryptos = [];
    },
  },
});
