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
        // Trier par rank croissant (1, 2, 3, 4...)
        return a.rank - b.rank;
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

        // S'abonner aux trades pour tous les symboles
        console.log("📡 Subscribing to trades (*) ...");
        rt.subscribe("trade", "*", "");

        // Handler pour les trades (prix en temps réel)
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

        // Handler pour les candles (statistiques mises à jour)
        const onCandle = (message: any) => {
          if (
            message.data &&
            message.data.symbol &&
            message.data.timeframe === "1m"
          ) {
            const symbol = message.data.symbol;
            const crypto = this.cryptoMap.get(symbol);

            if (crypto && message.data.close) {
              // Mettre à jour le prix avec la candle
              crypto.price = message.data.close;

              // Mettre à jour le volume si disponible
              if (message.data.volume) {
                crypto.volume24h = message.data.volume;
              }
            }
          }
        };

        // Écouter les trades et candles EN CONTINU
        rt.on("trade", onTrade);
        rt.on("candle", onCandle);

        // Découverte immédiate des cryptos, plus d'attente
        console.log(
          "✅ Discovered",
          this.cryptoMap.size,
          "cryptos from WebSocket"
        );

        // FALLBACK: Si aucun crypto découvert via WebSocket, charger depuis l'API
        if (this.cryptoMap.size === 0) {
          console.log(
            "🔄 No cryptos discovered via WebSocket, fetching from API..."
          );
          try {
            const response = await http.get<{
              success: boolean;
              data: { symbols: string[] };
            }>("/api/v1/crypto/symbols");

            if (
              response.data.success &&
              response.data.data?.symbols &&
              response.data.data.symbols.length > 0
            ) {
              console.log(
                `📥 Loaded ${response.data.data.symbols.length} symbols from API`
              );

              // Initialiser cryptoMap avec les symboles de la base de données
              for (const symbol of response.data.data.symbols) {
                if (!this.cryptoMap.has(symbol)) {
                  this.cryptoMap.set(symbol, {
                    id: symbol,
                    rank: this.cryptoMap.size + 1,
                    name: symbol.split("/")[0], // BTC de BTC/USDT
                    symbol,
                    price: 0,
                    change1h: 0,
                    change24h: 0,
                    change7d: 0,
                    marketCap: 0,
                    volume24h: 0,
                    circulatingSupply: 0,
                    sparklineData: [],
                  });
                }
              }

              console.log(
                `✅ Initialized ${this.cryptoMap.size} cryptos from database`
              );
            } else {
              console.warn("⚠️ No symbols returned from API");
            }
          } catch (error) {
            console.error("❌ Failed to load symbols from API:", error);
          }
        }

        // Maintenant charger les stats pour chaque crypto
        if (this.cryptoMap.size > 0) {
          console.log("📊 Loading stats for each crypto...");

          const symbols = Array.from(this.cryptoMap.keys());

          // Charger les stats en parallèle (avec throttling pour éviter de surcharger le serveur)
          const batchSize = 10;
          for (let i = 0; i < symbols.length; i += batchSize) {
            const batch = symbols.slice(i, i + batchSize);
            await Promise.all(
              batch.map(async (symbol) => {
                try {
                  const crypto = this.cryptoMap.get(symbol)!;

                  // Charger les données candles (dernières 24h avec interval 1h)
                  const candlesResponse = await http.get(
                    `/api/v1/crypto/data`,
                    {
                      params: {
                        symbol: symbol,
                        interval: "1h",
                        limit: 24,
                      },
                    }
                  );

                  const candles = candlesResponse.data?.data || [];
                  if (candles && candles.length > 0) {
                    // Calculer les stats à partir des candles
                    const latest = candles[0]; // Le plus récent
                    const oldest = candles[candles.length - 1];

                    // Mettre à jour le prix actuel
                    if (crypto.price === 0 && latest.close) {
                      crypto.price = latest.close;
                    }

                    // Calculer le changement 24h
                    if (oldest.close && latest.close) {
                      const priceChange =
                        ((latest.close - oldest.close) / oldest.close) * 100;
                      crypto.change24h = priceChange;
                      crypto.change1h = priceChange; // Approximation
                    }

                    // Calculer le volume 24h (somme des volumes)
                    crypto.volume24h = candles.reduce(
                      (sum: number, candle: any) => sum + (candle.volume || 0),
                      0
                    );

                    // Extraire les prix de clôture pour le sparkline (inverser pour avoir chronologique)
                    crypto.sparklineData = candles
                      .reverse()
                      .map((candle: any) => candle.close);
                  }

                  console.log("📈 Data loaded for", symbol);
                } catch (error) {
                  console.warn(
                    `⚠️ Could not load data for ${symbol}:`,
                    error
                  );
                  // Les stats restent à 0, ce n'est pas grave
                }
              })
            );
          }

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
