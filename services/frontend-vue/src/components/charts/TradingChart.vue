<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import CandleChart from "@/components/charts/CandleChart.vue";
import LineChart from "@/components/charts/LineChart.vue";
import { useMarketStore } from "@/stores/market";
import { useIndicatorsStore } from "@/stores/indicators";
import { fetchCandles } from "@/services/markets.api";
import {
  useTradingWebSocket,
  useLivePrices,
  useLiveCandles,
} from "@/services/websocket";
import type { CandleDTO } from "@/types/market";
import {
  TrendingUp,
  Settings,
  BarChart3,
  Minus,
  ExternalLink,
  ChevronDown,
} from "lucide-vue-next";

const store = useMarketStore();
const indicatorsStore = useIndicatorsStore();

// Debug: Afficher les changements dans la console
watch(
  () => indicatorsStore.showRSI,
  (newVal) => {
    console.log("RSI changed to:", newVal);
  }
);
watch(
  () => indicatorsStore.layoutMode,
  (newVal) => {
    console.log("Layout mode changed to:", newVal);
  }
);

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
    <div class="trading-chart-header">
      <div class="trading-chart-controls-left">
        <button
          :class="[
            'trading-chart-control-btn',
            { 'trading-chart-control-btn--active': chartMode === 'line' },
          ]"
          @click="chartMode = 'line'"
        >
          Line
        </button>
        <button
          :class="[
            'trading-chart-control-btn',
            'trading-chart-control-btn--secondary',
            { 'trading-chart-control-btn--active': chartMode === 'candle' },
          ]"
          @click="chartMode = 'candle'"
        >
          Candle
        </button>
        <button
          class="trading-chart-control-btn trading-chart-control-btn--icon"
        >
          <TrendingUp class="trading-chart-btn-icon" />
        </button>
        <button
          class="trading-chart-control-btn trading-chart-control-btn--icon"
        >
          <Settings class="trading-chart-btn-icon" />
        </button>
      </div>

      <div class="trading-chart-controls-center">
        <button class="trading-chart-trading-view-btn">
          <ExternalLink class="trading-chart-btn-icon" />
          TradingView
        </button>
        <button class="trading-chart-compare-btn">
          Comparer avec
          <ChevronDown class="trading-chart-btn-icon" />
        </button>
      </div>

      <div class="trading-chart-timeframes">
        <button
          v-for="tf in timeframes"
          :key="tf.value"
          :class="[
            'trading-chart-tf-btn',
            { 'trading-chart-tf-btn--active': selectedTimeframe === tf.value },
          ]"
          @click="changeTimeframe(tf.value)"
          :disabled="loading"
        >
          {{ tf.label }}
        </button>
        <button class="trading-chart-tf-btn">LOG</button>
        <button class="trading-chart-tf-btn">⋯</button>
      </div>
    </div>

    <!-- Chart container -->
    <div class="trading-chart-container">
      <div v-if="loading" class="trading-chart-loading">
        <div class="trading-chart-loading-spinner"></div>
        <span>Chargement des données...</span>
      </div>
      <template v-else>
        <!-- Navigation controls -->
        <div class="trading-chart-nav-controls">
          <button
            class="trading-chart-nav-btn"
            @click="navigateTime('prev')"
            title="Période précédente"
          >
            ◀
          </button>
          <button
            class="trading-chart-nav-btn trading-chart-nav-btn--zoom-out"
            @click="adjustZoom('out')"
            title="Dézoomer"
          >
            −
          </button>
          <button
            class="trading-chart-nav-btn trading-chart-nav-btn--zoom-in"
            @click="adjustZoom('in')"
            title="Zoomer"
          >
            +
          </button>
          <button
            class="trading-chart-nav-btn"
            @click="navigateTime('next')"
            title="Période suivante"
          >
            ▶
          </button>
          <button
            class="trading-chart-nav-btn trading-chart-nav-btn--reset"
            @click="resetChartView"
            title="Réinitialiser la vue"
          >
            ⌂
          </button>
        </div>

        <!-- Charts -->
        <div class="trading-chart-wrapper">
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

        <!-- Indicateurs techniques (mode compact uniquement) -->
        <div
          v-if="indicatorsStore.layoutMode === 'compact'"
          class="trading-chart-indicators"
        >
          <!-- Debug info -->
          <div style="font-size: 12px; color: #666; margin-bottom: 8px">
            Mode: {{ indicatorsStore.layoutMode }} | RSI:
            {{ indicatorsStore.showRSI }} | MACD: {{ indicatorsStore.showMACD }}
          </div>

          <!-- RSI Indicator -->
          <div v-if="indicatorsStore.showRSI" class="trading-chart-indicator">
            <div class="trading-chart-indicator-header">
              <span class="trading-chart-indicator-title"
                >RSI ({{ indicatorsStore.rsiPeriod }})</span
              >
              <span class="trading-chart-indicator-value">75.34</span>
            </div>
            <div class="trading-chart-indicator-chart">
              <div class="trading-chart-indicator-line rsi-line"></div>
              <div class="trading-chart-indicator-area rsi-area"></div>
            </div>
          </div>

          <!-- MACD Indicator -->
          <div v-if="indicatorsStore.showMACD" class="trading-chart-indicator">
            <div class="trading-chart-indicator-header">
              <span class="trading-chart-indicator-title"
                >MACD ({{ indicatorsStore.macdFast }},{{
                  indicatorsStore.macdSlow
                }},{{ indicatorsStore.macdSignal }})</span
              >
              <span class="trading-chart-indicator-value">0.0234</span>
            </div>
            <div class="trading-chart-indicator-chart">
              <div class="trading-chart-indicator-line macd-line"></div>
              <div
                class="trading-chart-indicator-histogram macd-histogram"
              ></div>
            </div>
          </div>

          <!-- Bollinger Bands Indicator -->
          <div
            v-if="indicatorsStore.showBollinger"
            class="trading-chart-indicator"
          >
            <div class="trading-chart-indicator-header">
              <span class="trading-chart-indicator-title"
                >Bollinger ({{ indicatorsStore.bbPeriod }},{{
                  indicatorsStore.bbStd
                }})</span
              >
              <span class="trading-chart-indicator-value">Upper: 45,234</span>
            </div>
            <div class="trading-chart-indicator-chart">
              <div class="trading-chart-indicator-bands bollinger-bands"></div>
            </div>
          </div>

          <!-- Momentum Indicator -->
          <div
            v-if="indicatorsStore.showMomentum"
            class="trading-chart-indicator"
          >
            <div class="trading-chart-indicator-header">
              <span class="trading-chart-indicator-title"
                >Momentum ({{ indicatorsStore.momPeriod }})</span
              >
              <span class="trading-chart-indicator-value">1.0234</span>
            </div>
            <div class="trading-chart-indicator-chart">
              <div class="trading-chart-indicator-line momentum-line"></div>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- Footer avec date/time comme CoinMarketCap -->
    <div class="trading-chart-footer"></div>
  </div>
</template>
