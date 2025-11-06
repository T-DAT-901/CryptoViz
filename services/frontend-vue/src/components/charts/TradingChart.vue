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

const { connect, disconnect, isConnected, lastUpdate } = useTradingWebSocket();
const { priceData: livePrice } = useLivePrices("BTCUSDT");
const { latestCandle, unsubscribe: unsubscribeCandles } = useLiveCandles();

const chartMode = ref<"candle" | "line">("line");
const loading = ref(false);
const candles = ref<CandleDTO[]>([]);

const lineChartRef = ref();
const candleChartRef = ref();

const rsiMiniChartRef = ref<HTMLCanvasElement | null>(null);
const macdMiniChartRef = ref<HTMLCanvasElement | null>(null);
let rsiMiniChart: Chart | null = null;
let macdMiniChart: Chart | null = null;

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

const selectedTimeframe = computed({
  get: () => indicatorsStore.selectedTimeframe,
  set: (value) => indicatorsStore.setTimeframe(value),
});

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
  return points;
});

// Load data based on selected timeframe
async function loadData() {
  loading.value = true;
  try {
    let rows = [];

    if (import.meta.env.VITE_USE_MOCK === "true") {
      const { default: unifiedData } = await import(
        "@/services/mocks/candles_unified.json"
      );

      switch (selectedTimeframe.value) {
        case "1h":
          const oneDayData = unifiedData["1d"] || [];
          rows = oneDayData.slice(-60);
          break;
        case "1d":
          const twentyFourHourData = unifiedData["1d"] || [];
          rows = twentyFourHourData.slice(-1440);
          break;
        case "7d":
          rows = unifiedData["7d"] || [];
          break;
        case "1M":
          rows = unifiedData["1M"] || [];
          break;
        case "1y":
          rows = unifiedData["1y"] || [];
          break;
        case "all":
          rows = unifiedData["all"] || [];
          break;
        default:
          const fallbackData = unifiedData["1d"] || [];
          rows = fallbackData.slice(-60);
      }
    } else {
      rows = await fetchCandles("BTC", selectedTimeframe.value, 500);
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
  if (candleData.length === 0) return;

  rsiMiniData.value = calculateRSIFromCandles(candleData);
  macdMiniData.value = calculateMACDFromCandles(candleData);

  await nextTick();
  buildMiniCharts();
}

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

function calculateMACDFromCandles(candles: any[]): Array<{
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
      timestamp: candles[i].t,
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

function buildMiniCharts() {
  setTimeout(() => {
    buildRSIMiniChart();
    buildMACDMiniChart();
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

async function changeTimeframe(
  newTimeframe: "1h" | "1d" | "7d" | "1M" | "1y" | "all"
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
  rsiMiniChart?.destroy();
  macdMiniChart?.destroy();
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
        <button
          class="trading-chart-control-btn trading-chart-control-btn--icon"
          @click="resetChartView"
          title="Reset chart view"
        >
          ⌂
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
    </div>

    <div class="trading-chart-footer"></div>
  </div>
</template>
