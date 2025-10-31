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

// Références pour accéder aux méthodes des charts enfants
const lineChartRef = ref();
const candleChartRef = ref();

// Timeframes disponibles (simplifiés et pertinents)
const timeframes = [
  { value: "1h", label: "1H" },
  { value: "1d", label: "24H" },
  { value: "7d", label: "7D" },
  { value: "1M", label: "1M" },
  { value: "1y", label: "1Y" },
  { value: "all", label: "ALL" },
] as const;

// Timeframe sélectionné
const selectedTimeframe = ref("1h");

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
        case "1h":
          // Pour 1H, prendre les 60 derniers points des données 1D (60 minutes)
          const oneDayData = unifiedData["1d"] || [];
          rows = oneDayData.slice(-60); // Dernière heure en minutes
          break;
        case "1d":
          // Pour 24H, prendre les dernières 24 heures des données 1D
          const twentyFourHourData = unifiedData["1d"] || [];
          rows = twentyFourHourData.slice(-1440); // Dernières 24 heures (1440 minutes)
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
          // Pour le fallback, utiliser 1h (dernière heure des données 1d)
          const fallbackData = unifiedData["1d"] || [];
          rows = fallbackData.slice(-60);
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

// Fonctions de navigation temporelle
function navigateTime(direction: "prev" | "next") {
  const activeChart =
    chartMode.value === "line" ? lineChartRef.value : candleChartRef.value;
  if (!activeChart?.chart) return;

  const chart = activeChart.chart;
  const xScale = chart.scales.x;
  const currentMin = xScale.min;
  const currentMax = xScale.max;
  const range = currentMax - currentMin;

  // Déplacer de 50% de la plage visible
  const shift = range * 0.5 * (direction === "next" ? 1 : -1);

  chart.zoomScale(
    "x",
    {
      min: currentMin + shift,
      max: currentMax + shift,
    },
    "default"
  );
}

// Ajuster le zoom
function adjustZoom(direction: "in" | "out") {
  const activeChart =
    chartMode.value === "line" ? lineChartRef.value : candleChartRef.value;
  if (!activeChart?.chart) return;

  const chart = activeChart.chart;
  const zoomFactor = direction === "in" ? 0.8 : 1.25; // Zoom de 20% in/out

  chart.zoom(zoomFactor);
}

// Réinitialiser la vue du chart
function resetChartView() {
  const activeChart =
    chartMode.value === "line" ? lineChartRef.value : candleChartRef.value;
  if (!activeChart?.resetZoom) return;

  activeChart.resetZoom();

  // Réajuster aux données après reset
  setTimeout(() => {
    if (activeChart?.fitChartToTimeframe) {
      activeChart.fitChartToTimeframe();
    }
  }, 100);
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

    <!-- Chart container -->
    <div class="chart-container">
      <div v-if="loading" class="loading">
        <div class="loading-spinner"></div>
        <span>Chargement des données...</span>
      </div>
      <template v-else>
        <!-- Navigation controls -->
        <div class="chart-nav-controls">
          <button
            class="nav-btn"
            @click="navigateTime('prev')"
            title="Période précédente"
          >
            ◀
          </button>
          <button
            class="nav-btn zoom-out"
            @click="adjustZoom('out')"
            title="Dézoomer"
          >
            −
          </button>
          <button
            class="nav-btn zoom-in"
            @click="adjustZoom('in')"
            title="Zoomer"
          >
            +
          </button>
          <button
            class="nav-btn"
            @click="navigateTime('next')"
            title="Période suivante"
          >
            ▶
          </button>
          <button
            class="nav-btn reset"
            @click="resetChartView"
            title="Réinitialiser la vue"
          >
            ⌂
          </button>
        </div>

        <!-- Charts -->
        <div class="chart-wrapper">
          <LineChart
            v-if="chartMode === 'line'"
            :points="linePoints"
            :timeframe="selectedTimeframe"
            ref="lineChartRef"
          />
          <CandleChart
            v-else-if="chartMode === 'candle'"
            :candles="candles"
            :timeframe="selectedTimeframe"
            ref="candleChartRef"
          />
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

.chart-container {
  position: relative;
  height: 400px;
  background: #040d12;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #9ca3af;
  font-size: 14px;
  gap: 12px;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid rgba(79, 70, 229, 0.2);
  border-top: 3px solid #4f46e5;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Contrôles de navigation du chart */
.chart-nav-controls {
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 10;
  display: flex;
  gap: 4px;
  background: rgba(4, 13, 18, 0.9);
  border-radius: 8px;
  padding: 4px;
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.nav-btn {
  background: rgba(26, 26, 46, 0.8);
  border: 1px solid #374151;
  border-radius: 4px;
  color: #9ca3af;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 12px;
  font-weight: 600;
}

.nav-btn:hover {
  background: #374151;
  color: #e5e7eb;
  transform: translateY(-1px);
}

.nav-btn:active {
  transform: translateY(0);
}

.nav-btn.zoom-in,
.nav-btn.zoom-out {
  font-size: 14px;
  font-weight: bold;
}

.nav-btn.reset {
  background: rgba(79, 70, 229, 0.2);
  border-color: #4f46e5;
  color: #4f46e5;
}

.nav-btn.reset:hover {
  background: #4f46e5;
  color: white;
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

  .chart-container {
    height: 300px;
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

  .chart-nav-controls {
    top: 8px;
    left: 8px;
    padding: 2px;
  }

  .nav-btn {
    width: 24px;
    height: 24px;
    font-size: 10px;
  }
}
</style>
