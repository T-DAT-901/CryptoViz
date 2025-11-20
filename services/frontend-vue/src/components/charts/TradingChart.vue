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
import { useRoute } from "vue-router";
import CandleChart from "@/components/charts/CandleChart.vue";
import LineChart from "@/components/charts/LineChart.vue";
import { useMarketStore } from "@/stores/market";
import { useIndicatorsStore } from "@/stores/indicators";
import { fetchCandles, fetchIndicators } from "@/services/markets.api";
import {
  useTradingWebSocket,
  useLivePrices,
  useLiveCandles,
  useLiveTrades,
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
const route = useRoute();

// Utiliser le symbolPair du Dashboard (ex: BTC/USDT)
const symbolPair = computed(() => {
  const symbol = (route.params.symbol as string) || "btc/usdt";
  // Décoder le symbole s'il est encodé en URL
  return decodeURIComponent(symbol).toUpperCase();
});

const { connect, disconnect, isConnected, lastUpdate } = useTradingWebSocket();
const { priceData: livePrice } = useLivePrices("BTCUSDT");
const { latestCandle, unsubscribe: unsubscribeCandles } = useLiveCandles();

// Get the symbol without / for WebSocket format (BTCUSDT instead of BTC/USDT)
const symbolForWS = computed(() => {
  return symbolPair.value.replace("/", "");
});

// Get live trades from WebSocket to populate the chart in real-time
const { trades: liveTrades, unsubscribe: unsubscribeTrades } = useLiveTrades(
  symbolForWS.value
);

const chartMode = ref<"candle" | "line">("line");
const loading = ref(false);
const candles = ref<CandleDTO[]>([]);

const lineChartRef = ref();
const candleChartRef = ref();

const rsiMiniChartRef = ref<HTMLCanvasElement | null>(null);
const macdMiniChartRef = ref<HTMLCanvasElement | null>(null);
const bollingerMiniChartRef = ref<HTMLCanvasElement | null>(null);
let rsiMiniChart: Chart | null = null;
let macdMiniChart: Chart | null = null;
let bollingerMiniChart: Chart | null = null;

const rsiMiniData = ref<Array<{ timestamp: number; value: number }>>([]);
const macdMiniData = ref<
  Array<{ timestamp: number; macd: number; signal: number; histogram: number }>
>([]);
const bollingerMiniData = ref<
  Array<{
    timestamp: number;
    upper: number;
    middle: number;
    lower: number;
  }>
>([]);

// Available timeframes - updated to granular intervals
const timeframes = [
  { value: "1m", label: "1m" },
  { value: "5m", label: "5m" },
  { value: "15m", label: "15m" },
  { value: "1h", label: "1h" },
  { value: "1d", label: "1d" },
] as const;

const selectedTimeframe = computed({
  get: () => indicatorsStore.selectedTimeframe,
  set: (value) => indicatorsStore.setTimeframe(value),
});

const linePoints = computed(() => {
  // Commencer par les points historiques du graph
  const historicalPoints = candles.value.map((c) => ({
    x: new Date(c.time).getTime(),
    y: c.close,
  }));

  // Ajouter les trades en temps réel du WebSocket
  const tradePoints = liveTrades.value.map((t) => ({
    x: t.timestamp,
    y: t.price,
  }));

  // Fusionner et trier par timestamp
  const allPoints = [...historicalPoints, ...tradePoints];
  allPoints.sort((a, b) => a.x - b.x);

  // Supprimer les doublons (basés sur la proximité temporelle)
  const uniquePoints = [];
  for (let i = 0; i < allPoints.length; i++) {
    if (i === 0 || allPoints[i].x - allPoints[i - 1].x > 100) {
      // Plus de 100ms d'écart
      uniquePoints.push(allPoints[i]);
    }
  }

  return uniquePoints;
});

// Load data based on selected timeframe
async function loadData() {
  loading.value = true;
  try {
    // Récupérer les données du backend réel avec le bon symbol et interval
    const symbol = symbolPair.value; // BTC/USDT, ETH/FDUSD, etc.
    const interval = selectedTimeframe.value; // 1m, 5m, 15m, 1h, 24h

    // Adapter le limit en fonction du timeframe pour avoir ~500-1000 points de données
    let limit = 500;
    if (interval === "1m")
      limit = 1440; // 1 jour de minutes
    else if (interval === "5m")
      limit = 288; // 1 jour de 5 minutes
    else if (interval === "15m")
      limit = 96; // 1 jour de 15 minutes
    else if (interval === "1h")
      limit = 168; // 1 semaine d'heures
    else if (interval === "1d") limit = 365; // 1 an de jours

    const rows = await fetchCandles(symbol, interval, limit);

    // Si pas de données, log un warning mais continue (la DB peut être vide)
    if (!rows || rows.length === 0) {
      console.warn(`Pas de données reçues pour ${symbol} - ${interval}`);
    }

    candles.value = rows;
    await loadIndicatorData(rows);
  } catch (error) {
    console.error("Error loading data:", error);
  } finally {
    loading.value = false;
  }
}

async function loadIndicatorData(candleData: any[]) {
  // Ne pas calculer les indicateurs si pas de données
  if (!candleData || candleData.length === 0) {
    console.log("🔴 No candle data, clearing all indicators");
    rsiMiniData.value = [];
    macdMiniData.value = [];
    bollingerMiniData.value = [];
    return;
  }

  try {
    // Charger les indicateurs depuis l'API au lieu de les calculer
    if (indicatorsStore.showRSI) {
      const rsiData = await fetchIndicators(symbolPair.value, "rsi");
      console.log("📊 RSI API response:", rsiData.length, "points");
      if (rsiData.length > 0) {
        rsiMiniData.value = rsiData;
      } else {
        console.log("❌ No RSI from API, not using fallback");
        rsiMiniData.value = [];
      }
    }

    if (indicatorsStore.showMACD) {
      const macdData = await fetchIndicators(symbolPair.value, "macd");
      console.log("📊 MACD API response:", macdData.length, "points");
      if (macdData.length > 0) {
        macdMiniData.value = macdData;
      } else {
        console.log("❌ No MACD from API, not using fallback");
        macdMiniData.value = [];
      }
    }

    if (indicatorsStore.showBollinger) {
      const bbData = await fetchIndicators(symbolPair.value, "bollinger");
      console.log("📊 Bollinger API response:", bbData.length, "points");
      if (bbData.length > 0) {
        bollingerMiniData.value = bbData;
      } else {
        console.log("❌ No Bollinger from API, not using fallback");
        bollingerMiniData.value = [];
      }
    }
  } catch (error) {
    console.warn("⚠️ Error loading indicators from API:", error);
    rsiMiniData.value = [];
    macdMiniData.value = [];
    bollingerMiniData.value = [];
  }

  await nextTick();
  buildMiniCharts();
}

function calculateRSIFromCandles(
  candles: any[]
): Array<{ timestamp: number; value: number }> {
  if (candles.length < 14) {
    return candles.map((candle, i) => ({
      timestamp: candle.time,
      value: 50 + Math.sin(i * 0.2) * 20,
    }));
  }

  const result: Array<{ timestamp: number; value: number }> = [];
  const rsiPeriod = indicatorsStore.rsiPeriod;

  for (let i = 0; i < candles.length; i++) {
    if (i < rsiPeriod - 1) {
      result.push({
        timestamp: new Date(candles[i].time).getTime(),
        value: 50 + Math.sin(i * 0.2) * 15,
      });
      continue;
    }

    let totalGains = 0;
    let totalLosses = 0;

    for (let j = i - rsiPeriod + 1; j <= i; j++) {
      if (j > 0) {
        const priceChange = candles[j].close - candles[j - 1].close;
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
      timestamp: new Date(candles[i].time).getTime(),
      value: rsi,
    });
  }

  return result;
}

function calculateMACDFromCandles(candles: any[]): Array<{
  timestamp: number;
  macd: number;
  signal: number;
  histogram: number;
}> {
  if (candles.length < 26) {
    return candles.map((candle, i) => ({
      timestamp: candle.time,
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
  const prices = candles.map((c) => c.close);

  const fastEMA = calculateSimpleEMA(prices, fastPeriod);
  const slowEMA = calculateSimpleEMA(prices, slowPeriod);

  for (let i = 0; i < candles.length; i++) {
    let macd = 0;
    let signal = 0;
    let histogram = 0;

    if (i >= slowPeriod - 1) {
      macd = fastEMA[i] - slowEMA[i];
      signal = macd * 0.8;
      histogram = macd - signal;
    }

    result.push({
      timestamp: new Date(candles[i].time).getTime(),
      macd: macd,
      signal: signal,
      histogram: histogram,
    });
  }

  return result;
}

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

function calculateBollingerFromCandles(candles: any[]): Array<{
  timestamp: number;
  upper: number;
  middle: number;
  lower: number;
}> {
  const period = 20;
  const stdDevMultiplier = 2;
  const result: Array<{
    timestamp: number;
    upper: number;
    middle: number;
    lower: number;
  }> = [];

  for (let i = period; i < candles.length; i++) {
    const slice = candles.slice(i - period, i);
    const prices = slice.map((c) => c.c);

    const sma = prices.reduce((sum, price) => sum + price, 0) / period;
    const variance =
      prices.reduce((sum, price) => sum + Math.pow(price - sma, 2), 0) / period;
    const stdDev = Math.sqrt(variance);

    result.push({
      timestamp: new Date(candles[i].time).getTime(),
      upper: sma + stdDevMultiplier * stdDev,
      middle: sma,
      lower: sma - stdDevMultiplier * stdDev,
    });
  }

  return result;
}

function buildMiniCharts() {
  setTimeout(() => {
    buildRSIMiniChart();
    buildMACDMiniChart();
    buildBollingerMiniChart();
  }, 100);
}

function buildRSIMiniChart() {
  if (!rsiMiniChartRef.value || rsiMiniData.value.length === 0) return;

  try {
    rsiMiniChart?.destroy();

    const rsiValues = rsiMiniData.value.map((d) => d.value);
    const currentRSI = rsiValues[rsiValues.length - 1] || 50;

    let mainColor = "#60a5fa";
    if (currentRSI >= 70) mainColor = "#ef4444";
    else if (currentRSI <= 30) mainColor = "#22c55e";

    rsiMiniChart = new Chart(rsiMiniChartRef.value, {
      type: "line",
      data: {
        labels: rsiValues.map((_, index) => index),
        datasets: [
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
          {
            data: new Array(rsiValues.length).fill(70),
            borderColor: "rgba(239, 68, 68, 0.6)",
            borderWidth: 1,
            borderDash: [2, 2],
            pointRadius: 0,
            fill: false,
          },
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
  } catch (error) {
    console.error("Error building RSI Mini Chart:", error);
  }
}

function buildMACDMiniChart() {
  if (!macdMiniChartRef.value || macdMiniData.value.length === 0) {
    return;
  }

  try {
    macdMiniChart?.destroy();

    const macdValues = macdMiniData.value.map((d) => d.macd);
    const histogramValues = macdMiniData.value.map((d) => d.histogram);
    const signalValues = macdMiniData.value.map((d) => d.signal);

    macdMiniChart = new Chart(macdMiniChartRef.value, {
      type: "line",
      data: {
        labels: macdValues.map((_, index) => index),
        datasets: [
          {
            type: "bar",
            data: histogramValues,
            backgroundColor: histogramValues.map((value) =>
              value >= 0 ? "rgba(34, 197, 94, 0.7)" : "rgba(239, 68, 68, 0.7)"
            ),
            borderWidth: 0,
            barThickness: 2,
          },
          {
            type: "line",
            data: macdValues,
            borderColor: "#3b82f6",
            borderWidth: 1.5,
            pointRadius: 0,
            fill: false,
            tension: 0.1,
          },
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
  } catch (error) {
    console.error("Error building MACD Mini Chart:", error);
  }
}

function buildBollingerMiniChart() {
  if (!bollingerMiniChartRef.value || bollingerMiniData.value.length === 0) {
    return;
  }

  try {
    bollingerMiniChart?.destroy();

    const upperValues = bollingerMiniData.value.map((d) => d.upper);
    const middleValues = bollingerMiniData.value.map((d) => d.middle);
    const lowerValues = bollingerMiniData.value.map((d) => d.lower);

    bollingerMiniChart = new Chart(bollingerMiniChartRef.value, {
      type: "line",
      data: {
        labels: middleValues.map((_, index) => index),
        datasets: [
          {
            data: upperValues,
            borderColor: "rgba(239, 68, 68, 0.6)",
            borderWidth: 1,
            pointRadius: 0,
            fill: false,
            tension: 0.2,
          },
          {
            data: middleValues,
            borderColor: "#3b82f6",
            borderWidth: 1.5,
            pointRadius: 0,
            fill: false,
            tension: 0.2,
          },
          {
            data: lowerValues,
            borderColor: "rgba(239, 68, 68, 0.6)",
            backgroundColor: "rgba(59, 130, 246, 0.08)",
            borderWidth: 1,
            pointRadius: 0,
            fill: "+1",
            tension: 0.2,
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
  } catch (error) {
    console.error("Error building Bollinger Mini Chart:", error);
  }
}

async function changeTimeframe(
  newTimeframe: "1m" | "5m" | "15m" | "1h" | "1d"
) {
  if (loading.value || selectedTimeframe.value === newTimeframe) return;
  selectedTimeframe.value = newTimeframe;
}

function resetChartView() {
  const activeChart =
    chartMode.value === "line" ? lineChartRef.value : candleChartRef.value;
  if (!activeChart?.resetZoom) return;

  activeChart.resetZoom();

  setTimeout(() => {
    if (activeChart?.fitChartToTimeframe) {
      activeChart.fitChartToTimeframe();
    }
  }, 100);
}

watch(latestCandle, (newCandle) => {
  if (newCandle && candles.value.length > 0) {
    const lastIndex = candles.value.length - 1;
    candles.value[lastIndex] = newCandle;
  }
});

watch(
  () => indicatorsStore.selectedTimeframe,
  async (newTimeframe) => {
    await loadData();
  }
);

watch(
  () => indicatorsStore.layoutMode,
  async (newMode) => {
    if (newMode === "compact") {
      await nextTick();
      setTimeout(() => {
        buildMiniCharts();
      }, 200);
    }
  }
);

onMounted(async () => {
  await loadData();
  try {
    await connect();
  } catch (error) {
    console.warn("WebSocket connection failed, using polling fallback");
  }
});

onUnmounted(() => {
  disconnect();
  unsubscribeCandles();
  unsubscribeTrades();
  rsiMiniChart?.destroy();
  macdMiniChart?.destroy();
  bollingerMiniChart?.destroy();
});
</script>

<template>
  <div class="trading-chart">
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
      </div>
    </div>

    <div class="trading-chart-container">
      <div v-if="loading" class="trading-chart-loading">
        <div class="trading-chart-loading-spinner"></div>
        <span>Loading data...</span>
      </div>
      <div v-else>
        <div class="trading-chart-nav-controls">
          <button
            class="trading-chart-nav-btn trading-chart-nav-btn--reset"
            @click="resetChartView"
            title="Reset view"
          >
            ⌂
          </button>
        </div>

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

        <div
          v-if="indicatorsStore.layoutMode === 'compact'"
          class="trading-chart-indicators"
        >
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
          <!-- Bollinger Indicator -->
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
            </div>
            <div class="trading-chart-indicator-chart">
              <canvas
                ref="bollingerMiniChartRef"
                class="trading-chart-mini-canvas"
              ></canvas>
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
    </div>

    <div class="trading-chart-footer"></div>
  </div>
</template>
