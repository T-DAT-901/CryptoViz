<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { Star, TrendingUp, TrendingDown } from "lucide-vue-next";
import { CryptoService, type CryptoData } from "@/services/crypto.api";

const router = useRouter();

// État de chargement et données
const loading = ref(true);
const cryptoList = ref<CryptoData[]>([]);

// Fonctions utilitaires
const formatPrice = (price: number): string => {
  return price.toLocaleString("fr-FR", {
    minimumFractionDigits: price >= 1 ? 2 : 4,
    maximumFractionDigits: price >= 1 ? 2 : 4,
  });
};

const formatLargeNumber = (num: number): string => {
  if (num >= 1e12) {
    return (num / 1e12).toFixed(1) + "T";
  } else if (num >= 1e9) {
    return (num / 1e9).toFixed(1) + "B";
  } else if (num >= 1e6) {
    return (num / 1e6).toFixed(1) + "M";
  } else if (num >= 1e3) {
    return (num / 1e3).toFixed(1) + "K";
  }
  return num.toString();
};

const getChangeClass = (change: number): string => {
  return change >= 0 ? "positive" : "negative";
};

const getChangeIcon = (change: number) => {
  return change >= 0 ? TrendingUp : TrendingDown;
};

// Actions
const toggleFavorite = (crypto: CryptoData) => {
  crypto.isFavorite = !crypto.isFavorite;
};

const navigateToChart = (crypto: CryptoData) => {
  router.push(`/dashboard/${crypto.symbol.toLowerCase()}`);
};

// Chargement des données au montage
const loadCryptos = async () => {
  try {
    loading.value = true;
    cryptoList.value = await CryptoService.getCryptoList();
    console.log(
      "Loaded",
      cryptoList.value.length,
      "cryptocurrencies from service"
    );
  } catch (error) {
    console.error("Error loading cryptocurrencies:", error);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  loadCryptos();
});
</script>

<template>
  <div class="home-page">
    <div class="page-header">
      <h1 class="page-title">Cryptomonnaies</h1>
      <p class="page-subtitle">
        Suivez les prix et performances des principales cryptomonnaies en temps
        réel
      </p>
    </div>

    <div class="crypto-table-container">
      <!-- État de chargement -->
      <div v-if="loading" class="loading-state">
        <div class="loading-spinner"></div>
        <p>Chargement des cryptomonnaies...</p>
      </div>

      <!-- Tableau des cryptos -->
      <table v-else-if="cryptoList.length > 0" class="crypto-table">
        <thead>
          <tr>
            <th class="col-rank">#</th>
            <th class="col-name">Nom</th>
            <th class="col-price">Prix</th>
            <th class="col-change">1h %</th>
            <th class="col-change">% 24h</th>
            <th class="col-change">% 7j</th>
            <th class="col-market-cap">Cap. Boursière</th>
            <th class="col-volume">Volume (24 h)</th>
            <th class="col-supply">Offre en Circulation</th>
            <th class="col-chart">7 Derniers Jours</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="crypto in cryptoList"
            :key="crypto.id"
            class="crypto-row"
            @click="navigateToChart(crypto)"
          >
            <!-- Favoris + Rank -->
            <td class="col-rank">
              <div class="rank-cell">
                <button
                  class="favorite-btn"
                  :class="{ active: crypto.isFavorite }"
                  @click.stop="toggleFavorite(crypto)"
                >
                  <Star :class="{ filled: crypto.isFavorite }" />
                </button>
                <span class="rank">{{ crypto.rank }}</span>
              </div>
            </td>

            <!-- Nom + Symbol -->
            <td class="col-name">
              <div class="name-cell">
                <div class="crypto-icon">
                  <img
                    :src="`https://cryptoicons.org/api/icon/${crypto.symbol.toLowerCase()}/32`"
                    :alt="crypto.name"
                    @error="
                      (e) =>
                        ((e.target as HTMLImageElement).style.display = 'none')
                    "
                  />
                </div>
                <div class="name-info">
                  <span class="crypto-name">{{ crypto.name }}</span>
                  <span class="crypto-symbol">{{ crypto.symbol }}</span>
                </div>
              </div>
            </td>

            <!-- Prix -->
            <td class="col-price">
              <span class="price">{{ formatPrice(crypto.price) }} €</span>
            </td>

            <!-- Changement 1h -->
            <td class="col-change">
              <div class="change-cell" :class="getChangeClass(crypto.change1h)">
                <component
                  :is="getChangeIcon(crypto.change1h)"
                  class="change-icon"
                />
                <span
                  >{{ crypto.change1h >= 0 ? "+" : ""
                  }}{{ crypto.change1h.toFixed(2) }}%</span
                >
              </div>
            </td>

            <!-- Changement 24h -->
            <td class="col-change">
              <div
                class="change-cell"
                :class="getChangeClass(crypto.change24h)"
              >
                <component
                  :is="getChangeIcon(crypto.change24h)"
                  class="change-icon"
                />
                <span
                  >{{ crypto.change24h >= 0 ? "+" : ""
                  }}{{ crypto.change24h.toFixed(2) }}%</span
                >
              </div>
            </td>

            <!-- Changement 7j -->
            <td class="col-change">
              <div class="change-cell" :class="getChangeClass(crypto.change7d)">
                <component
                  :is="getChangeIcon(crypto.change7d)"
                  class="change-icon"
                />
                <span
                  >{{ crypto.change7d >= 0 ? "+" : ""
                  }}{{ crypto.change7d.toFixed(2) }}%</span
                >
              </div>
            </td>

            <!-- Market Cap -->
            <td class="col-market-cap">
              <span class="market-cap"
                >€{{ formatLargeNumber(crypto.marketCap) }}</span
              >
            </td>

            <!-- Volume 24h -->
            <td class="col-volume">
              <div class="volume-cell">
                <span class="volume"
                  >€{{ formatLargeNumber(crypto.volume24h) }}</span
                >
              </div>
            </td>

            <!-- Circulating Supply -->
            <td class="col-supply">
              <span class="supply"
                >{{ formatLargeNumber(crypto.circulatingSupply) }}
                {{ crypto.symbol }}</span
              >
            </td>

            <!-- Mini Chart (Sparkline) -->
            <td class="col-chart">
              <div class="mini-chart">
                <svg width="120" height="40" viewBox="0 0 120 40">
                  <polyline
                    :points="
                      crypto.sparklineData
                        .map((price, index) => {
                          const x =
                            (index / (crypto.sparklineData.length - 1)) * 120;
                          const minPrice = Math.min(...crypto.sparklineData);
                          const maxPrice = Math.max(...crypto.sparklineData);
                          const y =
                            35 -
                            ((price - minPrice) / (maxPrice - minPrice)) * 30;
                          return `${x},${y}`;
                        })
                        .join(' ')
                    "
                    :class="getChangeClass(crypto.change7d)"
                    stroke-width="2"
                    fill="none"
                  />
                </svg>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Message si aucune donnée -->
      <div v-else class="no-data">
        <p>Aucune cryptomonnaie trouvée</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home-page {
  min-height: 100vh;
  background: #0a0e13;
  color: #e5e7eb;
  padding: 20px;
}

.page-header {
  margin-bottom: 30px;
  text-align: center;
}

.page-title {
  font-size: 32px;
  font-weight: 700;
  margin: 0 0 8px 0;
  color: #ffffff;
}

.page-subtitle {
  font-size: 16px;
  color: #9ca3af;
  margin: 0;
}

.crypto-table-container {
  background: #0b0e11;
  border-radius: 12px;
  border: 1px solid #1a1f24;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

/* États de chargement */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #9ca3af;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid rgba(79, 70, 229, 0.2);
  border-top: 3px solid #4f46e5;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.no-data {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #9ca3af;
  font-style: italic;
}

.crypto-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

.crypto-table thead {
  background: #141a1f;
  border-bottom: 1px solid #1a1f24;
}

.crypto-table th {
  padding: 16px 12px;
  text-align: left;
  font-weight: 600;
  color: #9ca3af;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.crypto-table tbody tr {
  border-bottom: 1px solid #1a1f24;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.crypto-table tbody tr:hover {
  background: #141a1f;
}

.crypto-table td {
  padding: 16px 12px;
  vertical-align: middle;
}

/* Colonnes spécifiques */
.col-rank {
  width: 80px;
}

.col-name {
  width: 200px;
}

.col-price {
  width: 120px;
}

.col-change {
  width: 100px;
}

.col-market-cap {
  width: 140px;
}

.col-volume {
  width: 140px;
}

.col-supply {
  width: 140px;
}

.col-chart {
  width: 140px;
}

/* Styles des cellules */
.rank-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.favorite-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #6b7280;
  transition: color 0.2s ease;
  padding: 4px;
}

.favorite-btn:hover {
  color: #fbbf24;
}

.favorite-btn.active {
  color: #fbbf24;
}

.favorite-btn .filled {
  fill: currentColor;
}

.rank {
  font-weight: 600;
  color: #9ca3af;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.crypto-icon {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  background: #1a1f24;
  display: flex;
  align-items: center;
  justify-content: center;
}

.crypto-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.name-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.crypto-name {
  font-weight: 600;
  color: #e5e7eb;
}

.crypto-symbol {
  font-size: 12px;
  color: #9ca3af;
  text-transform: uppercase;
}

.price {
  font-weight: 600;
  color: #e5e7eb;
  font-size: 15px;
}

.change-cell {
  display: flex;
  align-items: center;
  gap: 4px;
  font-weight: 600;
}

.change-cell.positive {
  color: #10b981;
}

.change-cell.negative {
  color: #ef4444;
}

.change-icon {
  width: 12px;
  height: 12px;
  stroke-width: 2;
}

.market-cap,
.volume,
.supply {
  font-weight: 500;
  color: #e5e7eb;
}

.mini-chart {
  width: 120px;
  height: 40px;
}

.mini-chart polyline.positive {
  stroke: #10b981;
}

.mini-chart polyline.negative {
  stroke: #ef4444;
}

/* Responsive */
@media (max-width: 1200px) {
  .col-supply,
  .col-volume {
    display: none;
  }
}

@media (max-width: 900px) {
  .col-market-cap,
  .col-chart {
    display: none;
  }
}

@media (max-width: 600px) {
  .col-change:nth-child(4),
  .col-change:nth-child(6) {
    display: none;
  }

  .crypto-table th,
  .crypto-table td {
    padding: 12px 8px;
  }
}
</style>
