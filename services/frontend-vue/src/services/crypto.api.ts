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
   * Écoute les trades en temps réel et extrait les symboles uniques
   */
  static async getAvailableCryptos(): Promise<CryptoData[]> {
    return new Promise(async (resolve) => {
      try {
        const rt = getRTClient();

        // Connecte si pas encore connecté
        if (!rt.isConnected()) {
          console.log("📡 Connecting to WebSocket...");
          await rt.connect();
        }

        console.log("📡 Subscribing to all trades via WebSocket...");

        // S'abonner aux trades de tous les symboles
        rt.subscribe("trade", "*");

        // Collecter les symboles uniques
        const symbols = new Set<string>();
        let unsubscribeHandler: (() => void) | null = null;

        const handler = (message: any) => {
          console.log("📥 Trade received:", message);
          if (message.data && message.data.symbol) {
            console.log("✨ Symbol collected:", message.data.symbol);
            symbols.add(message.data.symbol);
          }
        };

        // Enregistrer le handler pour les trades
        unsubscribeHandler = rt.on("trade", handler);

        // Attendre un peu pour collecter les symboles
        // puis se désabonner et retourner les résultats
        setTimeout(async () => {
          // Retirer le handler
          if (unsubscribeHandler) {
            unsubscribeHandler();
          }

          // Désabonner
          rt.unsubscribe("trade", "*");

          // Construire un tableau de CryptoData basique à partir des symboles
          const cryptos: CryptoData[] = Array.from(symbols).map(
            (symbol, index) => ({
              id: symbol,
              rank: index + 1,
              name: symbol.split("/")[0], // BTC de BTC/USDT
              symbol: symbol,
              price: 0,
              change1h: 0,
              change24h: 0,
              change7d: 0,
              marketCap: 0,
              volume24h: 0,
              circulatingSupply: 0,
              sparklineData: [],
            })
          );

          console.log("✅ Collected cryptos from WebSocket:", cryptos);

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
          } else {
            resolve(cryptos);
          }
        }, 5000); // Augmenté à 5 secondes pour collecter plus de trades
      } catch (error) {
        console.error(
          "❌ Error fetching available cryptos via WebSocket:",
          error
        );
        // Fallback sur les mocks si WebSocket échoue
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
