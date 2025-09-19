<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import CandleChart from "@/components/charts/CandleChart.vue";
import LineChart from "@/components/charts/LineChart.vue";
import { useMarketStore } from "@/stores/market";
import { fetchCandles } from "@/services/markets.api";
import {
  useTradingWebSocket,
  useLivePrices,
  useLiveCandles,
} from "@/services/websocket";
import type { CandleDTO } from "@/types/market";

const store = useMarketStore();

// WebSocket pour temps réel
const { connect, disconnect, isConnected, lastUpdate } = useTradingWebSocket();
const { priceData: livePrice } = useLivePrices("BTCUSDT");
const { latestCandle, unsubscribe: unsubscribeCandles } = useLiveCandles();

// Chart mode (candle ou line)
const chartMode = ref<"candle" | "line">("line");

// Loading state
const loading = ref(false);

// Données des bougies
const candles = ref<CandleDTO[]>([]);

// Timeframes disponibles (simplifiés et pertinents)
const timeframes = [
  { value: "1d", label: "1D" },
  { value: "7d", label: "7D" },
  { value: "1M", label: "1M" },
  { value: "1y", label: "1Y" },
  { value: "all", label: "ALL" },
] as const;

// Timeframe sélectionné
const selectedTimeframe = ref("7d");

// Données pour le LineChart
const linePoints = computed(() => {
  const points = candles.value.map((c) => ({
    x:
      typeof (c as any).ts === "number"
        ? (c as any).ts
        : typeof (c as any).t === "number"
          ? (c as any).t
          : new Date((c as any).t).getTime(),
    y: c.c,
  }));
  // console.log('TradingChart - linePoints:', candles.value.length, 'candles →', points.length, 'points'); // Debug
  return points;
});

// Prix affichés (combinant données statiques et WebSocket)
const displayPrice = computed(() => {
  if (livePrice.value) {
    return {
      date: new Date().toLocaleDateString("fr-FR"),
      time: new Date().toLocaleTimeString("fr-FR"),
      current: livePrice.value.price.toLocaleString("fr-FR") + " €",
      high: livePrice.value.high24h.toLocaleString("fr-FR") + " €",
      low: livePrice.value.low24h.toLocaleString("fr-FR") + " €",
      change: livePrice.value.change,
      changePercent: livePrice.value.changePercent,
      volume: (livePrice.value.volume / 1000000).toFixed(2) + "M €",
      isPositive: livePrice.value.change > 0,
    };
  } else if (candles.value.length > 0) {
    const latest = candles.value[candles.value.length - 1];
    const change = latest.c - latest.o;
    return {
      date: new Date().toLocaleDateString("fr-FR"),
      time: new Date().toLocaleTimeString("fr-FR"),
      current: latest.c.toLocaleString("fr-FR") + " €",
      high:
        Math.max(...candles.value.map((r) => r.h)).toLocaleString("fr-FR") +
        " €",
      low:
        Math.min(...candles.value.map((r) => r.l)).toLocaleString("fr-FR") +
        " €",
      change: change,
      changePercent: (change / latest.o) * 100,
      volume: "41.16M €",
      isPositive: change > 0,
    };
  }
  return null;
});

// Charge les données selon le timeframe
async function loadData() {
  loading.value = true;
  try {
    let rows = [];

    // Charger les données spécifiques au timeframe depuis le fichier unifié
    if (import.meta.env.VITE_USE_MOCK === "true") {
      console.log(
        `Loading mock data for timeframe: ${selectedTimeframe.value}`
      );

      // Importer les données unifiées
      const { default: unifiedData } = await import(
        "@/services/mocks/candles_unified.json"
      );

      // Sélectionner les données selon le timeframe
      switch (selectedTimeframe.value) {
        case "1d":
          rows = unifiedData["1d"] || []; // 1440 points (minutes)
          break;
        case "7d":
          rows = unifiedData["7d"] || []; // 168 points (heures)
          break;
        case "1M":
          rows = unifiedData["1M"] || []; // 180 points (4h)
          break;
        case "1y":
          rows = unifiedData["1y"] || []; // 365 points (jours)
          break;
        case "all":
          rows = unifiedData["all"] || []; // 520 points (semaines)
          break;
        default:
          rows = unifiedData["7d"] || []; // Fallback sur 7d
      }
    } else {
      // API réelle - adapter selon le timeframe
      rows = await fetchCandles("BTC", selectedTimeframe.value, 500);
    }

    candles.value = rows;
    console.log(
      `Loaded ${rows.length} candles for timeframe ${selectedTimeframe.value}`
    );
  } catch (error) {
    console.error("Error loading data:", error);
  } finally {
    loading.value = false;
  }
}

// Changer le timeframe avec feedback visuel
async function changeTimeframe(newTimeframe: string) {
  if (loading.value || selectedTimeframe.value === newTimeframe) return;

  selectedTimeframe.value = newTimeframe;
  await loadData();
}

// Mise à jour avec nouvelle bougie WebSocket
watch(latestCandle, (newCandle) => {
  if (newCandle && candles.value.length > 0) {
    // Remplacer la dernière bougie ou en ajouter une nouvelle
    const lastIndex = candles.value.length - 1;
    candles.value[lastIndex] = newCandle;
  }
});

// Lifecycle
onMounted(async () => {
  await loadData();
  // Connecter WebSocket après le chargement initial
  try {
    await connect();
  } catch (error) {
    console.warn("WebSocket connection failed, using polling fallback");
  }
});

onUnmounted(() => {
  disconnect();
  unsubscribeCandles();
});
</script>

<template>
  <div class="trading-chart">
    <!-- Header avec boutons Line/Candle et timeframes -->
    <div class="chart-header">
      <div class="chart-controls-left">
        <button
          :class="['control-btn', { active: chartMode === 'line' }]"
          @click="chartMode = 'line'"
        >
          Line
        </button>
        <button
          :class="[
            'control-btn',
            'secondary',
            { active: chartMode === 'candle' },
          ]"
          @click="chartMode = 'candle'"
        >
          Candle
        </button>
        <button class="control-btn icon-btn">📈</button>
        <button class="control-btn icon-btn">🔧</button>
      </div>

      <div class="chart-controls-center">
        <button class="trading-view-btn">📈 TradingView</button>
        <button class="compare-btn">Comparer avec ▼</button>
      </div>

      <div class="timeframes">
        <button
          v-for="tf in timeframes"
          :key="tf.value"
          :class="['tf-btn', { active: selectedTimeframe === tf.value }]"
          @click="changeTimeframe(tf.value)"
          :disabled="loading"
        >
          {{ tf.label }}
        </button>
        <button class="tf-btn">LOG</button>
        <button class="tf-btn">⋯</button>
      </div>
    </div>

    <!-- Prix overlay (style CoinMarketCap) -->
    <div class="price-overlay" v-if="!loading && displayPrice">
      <div class="price-info">
        <div class="date-time">
          {{ displayPrice.date }} {{ displayPrice.time }}
          <span class="timeframe-indicator">{{ selectedTimeframe }}</span>
        </div>
        <div class="price-details">
          <div class="price-item">
            <span class="label">{{
              chartMode === "candle" ? "Ouvrir:" : "Prix:"
            }}</span>
            <span class="value">{{ displayPrice.current }}</span>
          </div>
          <div class="price-item" v-if="chartMode === 'candle'">
            <span class="label">Haut:</span>
            <span class="value">{{ displayPrice.high }}</span>
          </div>
          <div class="price-item" v-if="chartMode === 'candle'">
            <span class="label">Bas:</span>
            <span class="value">{{ displayPrice.low }}</span>
          </div>
          <div class="price-item">
            <span class="label">Variation:</span>
            <span
              :class="[
                'value',
                displayPrice.isPositive ? 'positive' : 'negative',
              ]"
            >
              {{ displayPrice.change > 0 ? "+" : ""
              }}{{ displayPrice.change.toFixed(2) }} €({{
                displayPrice.changePercent.toFixed(2)
              }}%)
            </span>
          </div>
          <div class="price-item" v-if="chartMode === 'line'">
            <span class="label">Vol 24h:</span>
            <span class="value">{{ displayPrice.volume }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Chart container -->
    <div class="chart-container">
      <div v-if="loading" class="loading">Chargement des données...</div>
      <template v-else>
        <!-- Charts -->
        <div class="chart-wrapper">
          <LineChart v-if="chartMode === 'line'" :points="linePoints" />
          <CandleChart v-else-if="chartMode === 'candle'" :candles="candles" />
        </div>
      </template>
    </div>

    <!-- Footer avec date/time comme CoinMarketCap -->
    <div class="chart-footer"></div>
  </div>
</template>

<style scoped>
.trading-chart {
  background: #040d12;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
  border: 1px solid #1a1a2e;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px 8px;
  background: #040d12;
  border-bottom: 1px solid #16213e;
  flex-wrap: wrap;
  gap: 8px;
}

.chart-controls-left {
  display: flex;
  gap: 8px;
  align-items: center;
}

.chart-controls-center {
  display: flex;
  gap: 8px;
  align-items: center;
}

.control-btn {
  padding: 6px 12px;
  border-radius: 6px;
  background: #1a1a2e;
  border: 1px solid #2d2d44;
  color: #9ca3af;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 13px;
}

.control-btn:hover {
  background: #2d2d44;
  color: #e5e7eb;
}

.control-btn.active {
  background: #4f46e5;
  color: white;
  border-color: #4f46e5;
}

.control-btn.secondary {
  background: transparent;
  border-color: #374151;
}

.control-btn.icon-btn {
  padding: 6px 8px;
  min-width: 32px;
}

.trading-view-btn,
.compare-btn {
  padding: 6px 12px;
  border-radius: 6px;
  background: #1a1a2e;
  border: 1px solid #2d2d44;
  color: #9ca3af;
  cursor: pointer;
  font-size: 13px;
}

.timeframes {
  display: flex;
  gap: 4px;
  align-items: center;
  flex-wrap: wrap;
}

.tf-btn {
  padding: 4px 8px;
  border-radius: 4px;
  background: transparent;
  border: 1px solid transparent;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 12px;
  min-width: 28px;
  position: relative;
}

.tf-btn:hover:not(:disabled) {
  background: #1a1a2e;
  color: #9ca3af;
}

.tf-btn.active {
  background: #4f46e5;
  color: white;
  border-color: #4f46e5;
  box-shadow: 0 0 0 2px rgba(79, 70, 229, 0.2);
}

.tf-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.tf-btn:disabled.active::after {
  content: "";
  position: absolute;
  top: 50%;
  left: 50%;
  width: 12px;
  height: 12px;
  border: 2px solid transparent;
  border-top-color: white;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: translate(-50%, -50%) rotate(360deg);
  }
}

.price-overlay {
  position: absolute;
  top: 60px;
  left: 16px;
  z-index: 10;
  background: rgba(4, 13, 18, 0.85);
  backdrop-filter: blur(8px);
  border-radius: 8px;
  padding: 12px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.date-time {
  color: #e5e7eb;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.timeframe-indicator {
  background: #4f46e5;
  color: white;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.price-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.price-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.price-item .label {
  color: #9ca3af;
  font-size: 12px;
}

.price-item .value {
  color: #e5e7eb;
  font-size: 12px;
  font-weight: 500;
}

.price-item .value.positive {
  color: #10b981;
}

.price-item .value.negative {
  color: #ef4444;
}

.chart-container {
  position: relative;
  height: 400px;
  background: #040d12;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #9ca3af;
  font-size: 14px;
}

.chart-wrapper {
  height: 100%;
  width: 100%;
}

.chart-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #040d12;
  border-top: 1px solid #16213e;
}

.time-indicator {
  color: #6b7280;
  font-size: 11px;
  background: rgba(26, 26, 46, 0.8);
  padding: 4px 8px;
  border-radius: 4px;
}

.coinmarketcap-logo {
  color: #6b7280;
  font-size: 11px;
}

/* Responsive */
@media (max-width: 768px) {
  .chart-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
  }

  .chart-controls-left,
  .chart-controls-center {
    order: 2;
  }

  .timeframes {
    order: 1;
    justify-content: center;
  }

  .control-btn.secondary,
  .trading-view-btn,
  .compare-btn {
    display: none; /* Masquer sur mobile */
  }

  .price-overlay {
    position: relative;
    top: 0;
    left: 0;
    margin: 12px 16px;
    background: rgba(26, 26, 46, 0.95);
  }

  .chart-container {
    height: 300px;
  }

  .price-details {
    flex-direction: row;
    flex-wrap: wrap;
    gap: 8px;
  }

  .price-item {
    flex: 1;
    min-width: 120px;
  }
}

@media (max-width: 480px) {
  .timeframes {
    gap: 2px;
  }

  .tf-btn {
    padding: 3px 6px;
    font-size: 11px;
    min-width: 24px;
  }

  .chart-container {
    height: 250px;
  }

  .price-details {
    flex-direction: column;
  }
}
</style>
