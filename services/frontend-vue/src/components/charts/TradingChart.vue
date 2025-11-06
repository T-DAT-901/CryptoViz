<script setup lang="ts">
import {
  ref,
  computed,
  onMounted,
  onUnmounted,
  watch,
  nextTick,
  onBeforeUnmount,
} from "vue";
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
import {
  Chart,
  LineController,
  BarController,
  LineElement,
  BarElement,
  PointElement,
  LinearScale,
  CategoryScale,
  TimeScale,
} from "chart.js";

Chart.register(
  LineController,
  BarController,
  LineElement,
  BarElement,
  PointElement,
  LinearScale,
  CategoryScale,
  TimeScale
);

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

// WebSocket for real-time data
const { connect, disconnect, isConnected, lastUpdate } = useTradingWebSocket();
const { priceData: livePrice } = useLivePrices("BTCUSDT");
const { latestCandle, unsubscribe: unsubscribeCandles } = useLiveCandles();

// Chart mode (candle or line)
const chartMode = ref<"candle" | "line">("line");

// Loading state
const loading = ref(false);

// Candle data
const candles = ref<CandleDTO[]>([]);

// References to access child chart methods
const lineChartRef = ref();
const candleChartRef = ref();

// Mini-chart references
const rsiMiniChartRef = ref<HTMLCanvasElement | null>(null);
const macdMiniChartRef = ref<HTMLCanvasElement | null>(null);
let rsiMiniChart: Chart | null = null;
let macdMiniChart: Chart | null = null;

// Indicator data for mini-charts
const rsiMiniData = ref<Array<{ timestamp: number; value: number }>>([]);
const macdMiniData = ref<
  Array<{ timestamp: number; macd: number; signal: number; histogram: number }>
>([]);

// Available timeframes
const timeframes = [
  { value: "1h", label: "1H" },
  { value: "1d", label: "24H" },
  { value: "7d", label: "7D" },
  { value: "1M", label: "1M" },
  { value: "1y", label: "1Y" },
  { value: "all", label: "ALL" },
] as const;

// Selected timeframe (from store)
const selectedTimeframe = computed({
  get: () => indicatorsStore.selectedTimeframe,
  set: (value) => indicatorsStore.setTimeframe(value),
});

// Data for LineChart
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

// Load data based on selected timeframe
async function loadData() {
  loading.value = true;
  try {
    let rows = [];

    // Load timeframe-specific data from unified file
    if (import.meta.env.VITE_USE_MOCK === "true") {
      console.log(
        `Loading mock data for timeframe: ${selectedTimeframe.value}`
      );

      // Import unified data
      const { default: unifiedData } = await import(
        "@/services/mocks/candles_unified.json"
      );

      // Select data based on timeframe
      switch (selectedTimeframe.value) {
        case "1h":
          // For 1H, take last 60 points from 1D data (60 minutes)
          const oneDayData = unifiedData["1d"] || [];
          rows = oneDayData.slice(-60); // Last hour in minutes
          break;
        case "1d":
          // For 24H, take last 24 hours from 1D data
          const twentyFourHourData = unifiedData["1d"] || [];
          rows = twentyFourHourData.slice(-1440); // Last 24 hours (1440 minutes)
          break;
        case "7d":
          rows = unifiedData["7d"] || []; // 168 points (hours)
          break;
        case "1M":
          rows = unifiedData["1M"] || []; // 180 points (4h intervals)
          break;
        case "1y":
          rows = unifiedData["1y"] || []; // 365 points (days)
          break;
        case "all":
          rows = unifiedData["all"] || []; // 520 points (weeks)
          break;
        default:
          // For fallback, use 1h (last hour from 1d data)
          const fallbackData = unifiedData["1d"] || [];
          rows = fallbackData.slice(-60);
      }
    } else {
      // Real API - adapt based on timeframe
      rows = await fetchCandles("BTC", selectedTimeframe.value, 500);
    }

    candles.value = rows;
    console.log(
      `Loaded ${rows.length} candles for timeframe ${selectedTimeframe.value}`
    );

    // Load indicator data for mini-charts
    console.log("About to call loadIndicatorData with", rows.length, "candles");
    await loadIndicatorData(rows);
  } catch (error) {
    console.error("Error loading data:", error);
  } finally {
    loading.value = false;
  }
}

// Load indicator data for mini-charts
async function loadIndicatorData(candleData: any[]) {
  if (candleData.length === 0) return;

  // Calculate RSI
  rsiMiniData.value = calculateRSIFromCandles(candleData);

  // Calculate MACD
  macdMiniData.value = calculateMACDFromCandles(candleData);

  // Update mini-charts
  await nextTick();
  buildMiniCharts();
}

// Calculate RSI (simplified version for mini-charts)
function calculateRSIFromCandles(
  candles: any[]
): Array<{ timestamp: number; value: number }> {
  if (candles.length < 14) {
    return candles.map((candle, i) => ({
      timestamp: candle.t,
      value: 50 + Math.sin(i * 0.2) * 20,
    }));
  }

  const result: Array<{ timestamp: number; value: number }> = [];
  const rsiPeriod = indicatorsStore.rsiPeriod;

  for (let i = 0; i < candles.length; i++) {
    if (i < rsiPeriod - 1) {
      result.push({
        timestamp: candles[i].t,
        value: 50 + Math.sin(i * 0.2) * 15,
      });
      continue;
    }

    let totalGains = 0;
    let totalLosses = 0;

    for (let j = i - rsiPeriod + 1; j <= i; j++) {
      if (j > 0) {
        const priceChange = candles[j].c - candles[j - 1].c;
        if (priceChange > 0) {
          totalGains += priceChange;
        } else {
          totalLosses += Math.abs(priceChange);
        }
      }
    }

    const avgGain = totalGains / rsiPeriod;
    const avgLoss = totalLosses / rsiPeriod;

    let rsi = 50;
    if (avgLoss !== 0) {
      const rs = avgGain / avgLoss;
      rsi = 100 - 100 / (1 + rs);
    }

    rsi = Math.max(0, Math.min(100, rsi));

    result.push({
      timestamp: candles[i].t,
      value: rsi,
    });
  }

  return result;
}

// Calculate MACD (simplified version for mini-charts)
function calculateMACDFromCandles(
  candles: any[]
): Array<{
  timestamp: number;
  macd: number;
  signal: number;
  histogram: number;
}> {
  if (candles.length < 26) {
    return candles.map((candle, i) => ({
      timestamp: candle.t,
      macd: Math.sin(i * 0.1) * 2,
      signal: Math.sin(i * 0.1 - 0.3) * 1.5,
      histogram: Math.sin(i * 0.1) * 0.5,
    }));
  }

  const fastPeriod = indicatorsStore.macdFast;
  const slowPeriod = indicatorsStore.macdSlow;
  const signalPeriod = indicatorsStore.macdSignal;

  const result: Array<{
    timestamp: number;
    macd: number;
    signal: number;
    histogram: number;
  }> = [];
  const prices = candles.map((c) => c.c);

  // Calcul EMA simplifié
  const fastEMA = calculateSimpleEMA(prices, fastPeriod);
  const slowEMA = calculateSimpleEMA(prices, slowPeriod);

  for (let i = 0; i < candles.length; i++) {
    let macd = 0;
    let signal = 0;
    let histogram = 0;

    if (i >= slowPeriod - 1) {
      macd = fastEMA[i] - slowEMA[i];
      signal = macd * 0.8; // Approximation pour le mini-chart
      histogram = macd - signal;
    }

    result.push({
      timestamp: candles[i].t,
      macd: macd,
      signal: signal,
      histogram: histogram,
    });
  }

  return result;
}

// EMA simplifié pour les mini-charts
function calculateSimpleEMA(prices: number[], period: number): number[] {
  const ema: number[] = [];
  const multiplier = 2 / (period + 1);

  ema.push(prices[0] || 0);

  for (let i = 1; i < prices.length; i++) {
    const currentEMA: number =
      prices[i] * multiplier + ema[ema.length - 1] * (1 - multiplier);
    ema.push(currentEMA);
  }

  return ema;
}

// Build mini-charts
function buildMiniCharts() {
  // Wait a bit to ensure refs are available
  setTimeout(() => {
    buildRSIMiniChart();
    buildMACDMiniChart();
  }, 100);
}

// Build RSI mini-chart
function buildRSIMiniChart() {
  if (!rsiMiniChartRef.value || rsiMiniData.value.length === 0) return;

  try {
    rsiMiniChart?.destroy();

    // Use real calculated RSI data
    const rsiValues = rsiMiniData.value.map((d) => d.value);
    const currentRSI = rsiValues[rsiValues.length - 1] || 50;

    // Déterminer la couleur selon la zone RSI
    let mainColor = "#60a5fa"; // Bleu par défaut (zone neutre)
    if (currentRSI >= 70)
      mainColor = "#ef4444"; // Rouge (surachat)
    else if (currentRSI <= 30) mainColor = "#22c55e"; // Vert (survente)

    rsiMiniChart = new Chart(rsiMiniChartRef.value, {
      type: "line",
      data: {
        labels: rsiValues.map((_, index) => index),
        datasets: [
          // Ligne RSI principale
          {
            data: rsiValues,
            borderColor: mainColor,
            backgroundColor: mainColor
              .replace(")", ", 0.1)")
              .replace("#", "rgba(")
              .replace(/(.{2})(.{2})(.{2})/, "$1, $2, $3"),
            borderWidth: 2,
            pointRadius: 0,
            fill: true,
            tension: 0.1,
          },
          // Ligne 70 (surachat)
          {
            data: new Array(rsiValues.length).fill(70),
            borderColor: "rgba(239, 68, 68, 0.6)",
            borderWidth: 1,
            borderDash: [2, 2],
            pointRadius: 0,
            fill: false,
          },
          // Ligne 30 (survente)
          {
            data: new Array(rsiValues.length).fill(30),
            borderColor: "rgba(34, 197, 94, 0.6)",
            borderWidth: 1,
            borderDash: [2, 2],
            pointRadius: 0,
            fill: false,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: { display: false } },
        scales: {
          x: { display: false },
          y: {
            display: false,
            min: 0,
            max: 100,
          },
        },
        animation: false,
      },
    });

    console.log(
      "RSI Mini Chart built successfully! Current RSI:",
      currentRSI.toFixed(1)
    );
  } catch (error) {
    console.error("Error building RSI Mini Chart:", error);
  }
}

// Build MACD mini-chart
function buildMACDMiniChart() {
  if (!macdMiniChartRef.value || macdMiniData.value.length === 0) {
    console.log("MACD Mini Chart: Cannot build - missing ref or data", {
      hasRef: !!macdMiniChartRef.value,
      dataLength: macdMiniData.value.length,
    });
    return;
  }

  console.log(
    "Building MACD Mini Chart with",
    macdMiniData.value.length,
    "data points"
  );

  try {
    macdMiniChart?.destroy();

    // Utiliser les vraies données MACD calculées
    const macdValues = macdMiniData.value.map((d) => d.macd);
    const histogramValues = macdMiniData.value.map((d) => d.histogram);
    const signalValues = macdMiniData.value.map((d) => d.signal);

    macdMiniChart = new Chart(macdMiniChartRef.value, {
      type: "line",
      data: {
        labels: macdValues.map((_, index) => index),
        datasets: [
          // Histogramme MACD (barres colorées)
          {
            type: "bar",
            data: histogramValues,
            backgroundColor: histogramValues.map((value) =>
              value >= 0 ? "rgba(34, 197, 94, 0.7)" : "rgba(239, 68, 68, 0.7)"
            ),
            borderWidth: 0,
            barThickness: 2,
          },
          // Ligne MACD
          {
            type: "line",
            data: macdValues,
            borderColor: "#3b82f6",
            borderWidth: 1.5,
            pointRadius: 0,
            fill: false,
            tension: 0.1,
          },
          // Ligne Signal
          {
            type: "line",
            data: signalValues,
            borderColor: "#f59e0b",
            borderWidth: 1,
            pointRadius: 0,
            fill: false,
            tension: 0.1,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: { display: false } },
        scales: {
          x: { display: false },
          y: { display: false },
        },
        animation: false,
      },
    });

    console.log("MACD Mini Chart built successfully with real data!");
  } catch (error) {
    console.error("Error building MACD Mini Chart:", error);
  }
}

// Fonction debug pour tester les mini-charts
function debugMiniCharts() {
  console.log("=== DEBUG MINI-CHARTS ===");
  console.log("RSI data:", rsiMiniData.value.length, "points");
  console.log("MACD data:", macdMiniData.value.length, "points");
  console.log("RSI ref:", !!rsiMiniChartRef.value);
  console.log("MACD ref:", !!macdMiniChartRef.value);
  console.log("Layout mode:", indicatorsStore.layoutMode);
  console.log("Show RSI:", indicatorsStore.showRSI);
  console.log("Show MACD:", indicatorsStore.showMACD);

  if (rsiMiniData.value.length > 0) {
    console.log("First RSI point:", rsiMiniData.value[0]);
    console.log(
      "Last RSI point:",
      rsiMiniData.value[rsiMiniData.value.length - 1]
    );
  }

  if (macdMiniData.value.length > 0) {
    console.log("First MACD point:", macdMiniData.value[0]);
    console.log(
      "Last MACD point:",
      macdMiniData.value[macdMiniData.value.length - 1]
    );
  }

  // Forcer la reconstruction
  buildMiniCharts();
}

// Changer le timeframe avec feedback visuel
async function changeTimeframe(
  newTimeframe: "1h" | "1d" | "7d" | "1M" | "1y" | "all"
) {
  if (loading.value || selectedTimeframe.value === newTimeframe) return;

  // Le watcher va automatiquement recharger les données
  selectedTimeframe.value = newTimeframe;
}

// Time navigation functions
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

// Watcher pour recharger les données quand le timeframe change
watch(
  () => indicatorsStore.selectedTimeframe,
  async (newTimeframe) => {
    console.log("TradingChart: Timeframe changed to", newTimeframe);
    await loadData();
  }
);

// Watcher pour reconstruire les mini-charts quand le mode layout change
watch(
  () => indicatorsStore.layoutMode,
  async (newMode) => {
    console.log("TradingChart: Layout mode changed to", newMode);
    if (newMode === "compact") {
      // Attendre que le DOM soit mis à jour
      await nextTick();
      setTimeout(() => {
        console.log("Rebuilding mini-charts after layout change...");
        buildMiniCharts();
      }, 200);
    }
  }
);

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
  // Nettoyer les mini-charts
  rsiMiniChart?.destroy();
  macdMiniChart?.destroy();
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
      <div v-else>
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
          <!-- Debug info (peut être supprimé en production) -->
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
              <span class="trading-chart-indicator-value">
                {{
                  rsiMiniData.length > 0
                    ? rsiMiniData[rsiMiniData.length - 1]?.value?.toFixed(1)
                    : "--"
                }}
              </span>
            </div>
            <div class="trading-chart-indicator-chart">
              <canvas
                ref="rsiMiniChartRef"
                class="trading-chart-mini-canvas"
              ></canvas>
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
              <span class="trading-chart-indicator-value">
                {{
                  macdMiniData.length > 0
                    ? macdMiniData[macdMiniData.length - 1]?.macd?.toFixed(4)
                    : "--"
                }}
              </span>
            </div>
            <div class="trading-chart-indicator-chart">
              <canvas
                ref="macdMiniChartRef"
                class="trading-chart-mini-canvas"
              ></canvas>
            </div>
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
    </div>

    <!-- Footer avec date/time comme CoinMarketCap -->
    <div class="trading-chart-footer"></div>
  </div>
</template>
