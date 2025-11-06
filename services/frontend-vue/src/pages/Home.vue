<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { TrendingUp, TrendingDown } from "lucide-vue-next";
import { CryptoService, type CryptoData } from "@/services/crypto.api";

const router = useRouter();

const loading = ref(true);
const cryptoList = ref<CryptoData[]>([]);

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

const navigateToChart = (crypto: CryptoData) => {
  router.push(`/dashboard/${crypto.symbol.toLowerCase()}`);
};

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
      <div class="header-content">
        <div class="header-left">
          <h1 class="page-title">Market Overview</h1>
          <p class="page-subtitle">
            Real-time cryptocurrency prices and market data
          </p>
        </div>
        <div class="header-right">
          <div class="market-stats">
            <div class="stat-item">
              <span class="stat-label">24h Volume</span>
              <span class="stat-value">€2.1T</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">Market Cap</span>
              <span class="stat-value">€3.2T</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="crypto-table-container">
      <div v-if="loading" class="loading-state">
        <div class="loading-spinner"></div>
        <p>Chargement des cryptomonnaies...</p>
      </div>

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
            <td class="col-rank">
              <div class="rank-cell">
                <span class="rank">{{ crypto.rank }}</span>
              </div>
            </td>

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

      <div v-else class="no-data">
        <p>Aucune cryptomonnaie trouvée</p>
      </div>
    </div>
  </div>
</template>
