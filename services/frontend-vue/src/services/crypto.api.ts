import cryptosData from "./mocks/cryptos.json";

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

/**
 * Service pour récupérer les données des cryptomonnaies
 * Pour l'instant utilise des données mockées, mais peut être étendu pour appeler une vraie API
 */
export class CryptoService {
  /**
   * Récupère la liste complète des cryptomonnaies
   */
  static async getCryptoList(): Promise<CryptoData[]> {
    // Simulation d'un délai d'API
    await new Promise((resolve) => setTimeout(resolve, 100));

    return cryptosData as CryptoData[];
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
