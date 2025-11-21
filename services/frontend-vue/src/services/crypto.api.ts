// Types pour les données crypto
export interface CryptoData {
  id: string;
  rank: number;
  name: string;
  symbol: string;
  price: number;
  change1h: number;
  change24h: number;
  change7d: number;
  marketCap: number;
  volume24h: number;
  circulatingSupply: number;
  sparklineData: number[];
  isFavorite?: boolean;
}

import { getRTClient } from "./rt";
import { http } from "./http";

/**
 * Service pour récupérer les données des cryptomonnaies
 */
export class CryptoService {
  /**
   * Récupère la liste des cryptos disponibles via WebSocket
   * Prices depuis /ws/crypto (trades)
   * Stats depuis l'API REST (WebSocket stats non dispo)
   */
  static async getAvailableCryptos(): Promise<CryptoData[]> {
    return new Promise(async (resolve) => {
      try {
        const rt = getRTClient();

        // Connecte si pas encore connecté
        if (!rt.isConnected()) {
          console.log("📡 Connecting to Trades WebSocket...");
          await rt.connect();
        }

        console.log("📡 Subscribing to trades via /ws/crypto...");

        // S'abonner SEULEMENT aux trades (stats pas dispo sur WebSocket)
        rt.subscribe("trade", "*");

        // Collecter les symboles et prix
        const pricesMap = new Map<string, number>();
        let tradeHandler: (() => void) | null = null;

        // Handler pour les trades (prix)
        const onTrade = (message: any) => {
          if (message.data && message.data.symbol && message.data.price) {
            const symbol = message.data.symbol;
            console.log("✨ Trade:", symbol, "@", message.data.price);
            pricesMap.set(symbol, message.data.price);
          }
        };

        // Enregistrer le handler
        tradeHandler = rt.on("trade", onTrade);

        // Attendre pour collecter les données
        setTimeout(async () => {
          // Retirer le handler
          if (tradeHandler) tradeHandler();

          // Désabonner
          rt.unsubscribe("trade", "*");

          console.log("📥 Collected prices from trades:", pricesMap);

          // Construire un tableau de CryptoData avec les prix du WebSocket
          let cryptos: CryptoData[] = Array.from(pricesMap).map(
            ([symbol, price], index) => ({
              id: symbol,
              rank: index + 1,
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
            })
          );

          console.log("✅ Collected cryptos with prices:", cryptos);

          // Si aucune crypto collectée, fallback sur les mocks
          if (cryptos.length === 0) {
            console.log("⚠️ No cryptos collected, falling back to mocks...");
            try {
              const cryptosData = await import("./mocks/cryptos.json");
              console.log(
                "✅ Loaded mock cryptos:",
                cryptosData.default.length,
                "items"
              );
              resolve(cryptosData.default as CryptoData[]);
            } catch (mockError) {
              console.error("❌ Error loading mock cryptos:", mockError);
              resolve([]);
            }
            return;
          }

          // Enrichir avec les stats de l'API REST pour chaque crypto
          console.log("📊 Fetching stats from API for each symbol...");
          const enrichedCryptos = await Promise.all(
            cryptos.map(async (crypto) => {
              try {
                // Backend-go uses query parameters: /api/v1/stats?symbol=BTC/USDT
                const response = await http.get(`/api/v1/stats`, {
                  params: { symbol: crypto.symbol },
                });
                const stats = response.data?.data || {};

                return {
                  ...crypto,
                  change1h: stats.change1h || 0,
                  change24h: stats.change24h || 0,
                  change7d: stats.change7d || 0,
                  marketCap: stats.marketCap || 0,
                  volume24h: stats.volume24h || stats.total_volume || 0,
                  circulatingSupply: stats.circulatingSupply || 0,
                };
              } catch (error) {
                console.warn(
                  `⚠️ Could not fetch stats for ${crypto.symbol}:`,
                  error
                );
                return crypto; // Retourner les données partielles
              }
            })
          );

          console.log("✅ Enriched cryptos with stats:", enrichedCryptos);
          resolve(enrichedCryptos);
        }, 5000); // Attendre 5 secondes
      } catch (error) {
        console.error(
          "❌ Error fetching available cryptos via WebSocket:",
          error
        );
        console.log("📦 Falling back to mock cryptos...");
        try {
          const cryptosData = await import("./mocks/cryptos.json");
          console.log(
            "✅ Loaded mock cryptos:",
            cryptosData.default.length,
            "items"
          );
          resolve(cryptosData.default as CryptoData[]);
        } catch (mockError) {
          console.error("❌ Error loading mock cryptos:", mockError);
          resolve([]);
        }
      }
    });
  }

  /**
   * Récupère la liste complète des cryptomonnaies
   */
  static async getCryptoList(): Promise<CryptoData[]> {
    // Simulation d'un délai d'API
    await new Promise((resolve) => setTimeout(resolve, 100));

    const cryptosData = await import("./mocks/cryptos.json");
    return cryptosData.default as CryptoData[];
  }

  /**
   * Récupère une cryptomonnaie spécifique par son symbole
   */
  static async getCrypto(symbol: string): Promise<CryptoData | null> {
    const cryptos = await this.getCryptoList();
    return (
      cryptos.find(
        (crypto) => crypto.symbol.toLowerCase() === symbol.toLowerCase()
      ) || null
    );
  }

  /**
   * Recherche des cryptomonnaies par nom ou symbole
   */
  static async searchCryptos(query: string): Promise<CryptoData[]> {
    const cryptos = await this.getCryptoList();
    const searchTerm = query.toLowerCase();

    return cryptos.filter(
      (crypto) =>
        crypto.name.toLowerCase().includes(searchTerm) ||
        crypto.symbol.toLowerCase().includes(searchTerm)
    );
  }

  /**
   * Récupère les cryptomonnaies favorites
   */
  static async getFavoriteCryptos(): Promise<CryptoData[]> {
    const cryptos = await this.getCryptoList();
    return cryptos.filter((crypto) => crypto.isFavorite);
  }
}
