import { defineStore } from "pinia";
import { getRTClient } from "@/services/rt";
import { http } from "@/services/http";
import type { CryptoData } from "@/services/crypto.api";

export const useCryptoStore = defineStore("crypto", {
  state: () => ({
    cryptoMap: new Map<string, CryptoData>(),
    loading: false,
    rtConnected: false,
  }),

  getters: {
    sortedCryptos(): CryptoData[] {
      return Array.from(this.cryptoMap.values()).sort((a, b) => {
        // Trier par volume 24h décroissant (plus gros volume en premier)
        if (b.volume24h !== a.volume24h) {
          return b.volume24h - a.volume24h;
        }
        // Si même volume, trier par prix décroissant
        return b.price - a.price;
      });
    },
  },

  actions: {
    /**
     * Connecte au WebSocket, collecte les trades (prix)
     * Puis charge les stats pour chaque symbole via API REST
     */
    async initializeFromWebSocket() {
      try {
        this.loading = true;
        const rt = getRTClient();

        // Connecte si pas encore connecté
        if (!rt.isConnected()) {
          console.log("📡 Connecting to WebSocket...");
          await rt.connect();
        }

        this.rtConnected = true;

        // S'abonner aux trades
        console.log("📡 Subscribing to trades (*) ...");
        rt.subscribe("trade", "*");

        // Collecter les trades pendant 5 secondes
        const onTrade = (message: any) => {
          if (message.data && message.data.symbol && message.data.price) {
            const symbol = message.data.symbol;
            const price = message.data.price;

            if (!this.cryptoMap.has(symbol)) {
              // Nouveau symbole trouvé
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
              console.log("✨ Found crypto:", symbol, "@", price);
            } else {
              // Mettre à jour le prix
              const crypto = this.cryptoMap.get(symbol)!;
              crypto.price = price;
            }
          }
        };

        // Écouter les trades EN CONTINU pour mettre à jour les prix
        rt.on("trade", onTrade);

        // Attendre 5 secondes pour découvrir les cryptos
        console.log("⏳ Discovering cryptos for 5 seconds...");
        await new Promise((resolve) => setTimeout(resolve, 5000));

        console.log(
          "✅ Discovered",
          this.cryptoMap.size,
          "cryptos from WebSocket"
        );

        // Maintenant charger les stats pour chaque crypto
        if (this.cryptoMap.size > 0) {
          console.log("📊 Loading stats for each crypto...");

          const symbols = Array.from(this.cryptoMap.keys());

          // Charger les stats en parallèle
          await Promise.all(
            symbols.map(async (symbol) => {
              try {
                const response = await http.get(`/api/v1/stats/${symbol}`);
                const stats = response.data?.data || {};

                const crypto = this.cryptoMap.get(symbol)!;
                if (stats) {
                  crypto.change1h = stats.change1h || 0;
                  crypto.change24h = stats.change24h || 0;
                  crypto.change7d = stats.change7d || 0;
                  crypto.marketCap = stats.marketCap || 0;
                  crypto.volume24h = stats.volume24h || 0;
                  crypto.circulatingSupply = stats.circulatingSupply || 0;
                }

                console.log("📈 Stats loaded for", symbol);
              } catch (error) {
                console.warn(`⚠️ Could not load stats for ${symbol}:`, error);
                // Les stats restent à 0, ce n'est pas grave
              }
            })
          );

          console.log("✅ All stats loaded!");
        }

        this.loading = false;
      } catch (error) {
        console.error("❌ Error initializing crypto store:", error);
        this.loading = false;

        // Fallback sur mocks
        try {
          const cryptosData = await import("@/services/mocks/cryptos.json");
          this.cryptoMap.clear();
          cryptosData.default.forEach((crypto: CryptoData) => {
            this.cryptoMap.set(crypto.symbol, crypto);
          });
          console.log("✅ Loaded mock cryptos");
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
    },
  },
});
