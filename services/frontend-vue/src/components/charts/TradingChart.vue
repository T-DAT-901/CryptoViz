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
import VolumeChart from "@/components/charts/VolumeChart.vue";
import { useMarketStore } from "@/stores/market";
import { useIndicatorsStore } from "@/stores/indicators";
import { fetchCandles } from "@/services/markets.api";
import { fetchRSI, fetchMACD, fetchBollinger, fetchMomentum } from "@/services/indicators.api";
import {
  useTradingWebSocket,
  useLivePrices,
  useLiveCandles,
  useLiveTrades,
} from "@/services/websocket";
import { RealtimeCandleBuilder } from "@/services/rt";
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

// Progressive loading configuration from environment variables
const INITIAL_CANDLES = Number(import.meta.env.VITE_CHART_INITIAL_CANDLES) || 500;
const DEFAULT_ZOOM_CANDLES = Number(import.meta.env.VITE_CHART_DEFAULT_ZOOM_CANDLES) || 48;
const AUTOFETCH_THRESHOLD = Number(import.meta.env.VITE_CHART_AUTOFETCH_THRESHOLD) || 20;
const FETCH_BATCH_SIZE = Number(import.meta.env.VITE_CHART_FETCH_BATCH_SIZE) || 500;

// Progressive loading state management
const loadedTimeRange = ref<{ earliest: Date | null; latest: Date | null }>({
  earliest: null,
  latest: null,
});
const isLoadingMore = ref(false);
const hasMoreHistoricalData = ref(true);

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

// Real-time candle builder for 1m timeframe
const candleBuilder = ref<RealtimeCandleBuilder | null>(null);
const currentBuildingCandle = ref<{
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
} | null>(null);
let lastProcessedTradeIndex = 0;
let tradeProcessingInterval: ReturnType<typeof setInterval> | null = null;

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

// Computed qui force Vue à détecter les changements du building candle
// en créant une nouvelle référence à chaque fois
const buildingCandleForChart = computed(() => {
  if (selectedTimeframe.value === "1m" && currentBuildingCandle.value) {
    const candle = { ...currentBuildingCandle.value };
    console.log("📤 TradingChart: buildingCandleForChart computed", {
      timeframe: selectedTimeframe.value,
      candle,
    });
    return candle;
  }
  return null;
});

const linePoints = computed(() => {
  // Commencer par les points historiques du graph
  const historicalPoints = candles.value.map((c) => ({
    x: new Date(c.time).getTime(),
    y: c.close,
  }));

  // Ajouter les trades en temps réel du WebSocket
  let tradePoints = liveTrades.value.map((t) => ({
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

// Volume data computed from candles
const volumeData = computed(() => {
  return candles.value.map((candle) => ({
    timestamp: new Date(candle.time).getTime(),
    volume: candle.volume,
    color: candle.close >= candle.open ? "green" : "red",
  }));
});

// Load data based on selected timeframe
async function loadData() {
  loading.value = true;
  try {
    // Récupérer les données du backend réel avec le bon symbol et interval
    const symbol = symbolPair.value; // BTC/USDT, ETH/FDUSD, etc.
    const interval = selectedTimeframe.value; // 1m, 5m, 15m, 1h, 24h

    // Progressive loading: load only INITIAL_CANDLES most recent candles
    // This prevents Chart.js crashes and improves performance
    const limit = INITIAL_CANDLES;
    console.log(`📊 Loading ${limit} most recent candles for ${symbol} ${interval}`);

    const rows = await fetchCandles(symbol, interval, limit);

    // Si pas de données, log un warning mais continue (la DB peut être vide)
    if (!rows || rows.length === 0) {
      console.warn(`Pas de données reçues pour ${symbol} - ${interval}`);
      loadedTimeRange.value = { earliest: null, latest: null };
      hasMoreHistoricalData.value = false;
    } else {
      // Track loaded time range for progressive loading
      loadedTimeRange.value = {
        earliest: new Date(rows[0].time),
        latest: new Date(rows[rows.length - 1].time),
      };
      hasMoreHistoricalData.value = rows.length === limit; // If we got fewer than requested, no more data
      console.log(
        `📊 Loaded ${rows.length} candles. Range: ${loadedTimeRange.value.earliest?.toISOString()} to ${loadedTimeRange.value.latest?.toISOString()}`
      );
    }

    candles.value = rows;
    await loadIndicatorData(rows);

    // Set default zoom level after data loads
    await nextTick();
    setDefaultZoom();
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
      const rawRsiData = await fetchRSI(
        symbolPair.value,
        selectedTimeframe.value,
        indicatorsStore.rsiPeriod,
        10000
      );
      console.log("📊 RSI API response:", rawRsiData.length, "points");
      if (rawRsiData.length > 0) {
        // Transform backend format to chart format
        rsiMiniData.value = rawRsiData.map((item: any) => ({
          timestamp: new Date(item.time).getTime(),
          value: item.value || 0,
        }));
      } else {
        console.log("❌ No RSI from API, not using fallback");
        rsiMiniData.value = [];
      }
    }

    if (indicatorsStore.showMACD) {
      const rawMacdData = await fetchMACD(
        symbolPair.value,
        selectedTimeframe.value,
        indicatorsStore.macdFast,
        indicatorsStore.macdSlow,
        indicatorsStore.macdSignal,
        10000
      );
      console.log("📊 MACD API response:", rawMacdData.length, "points");
      if (rawMacdData.length > 0) {
        // Transform backend format to chart format
        macdMiniData.value = rawMacdData.map((item: any) => ({
          timestamp: new Date(item.time).getTime(),
          macd: item.value || 0,
          signal: item.value_signal || 0,
          histogram: item.value_histogram || 0,
        }));
      } else {
        console.log("❌ No MACD from API, not using fallback");
        macdMiniData.value = [];
      }
    }

    if (indicatorsStore.showBollinger) {
      const rawBbData = await fetchBollinger(
        symbolPair.value,
        selectedTimeframe.value,
        indicatorsStore.bbPeriod,
        indicatorsStore.bbStd,
        10000
      );
      console.log("📊 Bollinger API response:", rawBbData.length, "points");
      if (rawBbData.length > 0) {
        // Transform backend format to chart format
        bollingerMiniData.value = rawBbData.map((item: any) => ({
          timestamp: new Date(item.time).getTime(),
          upper: item.upper_band || 0,
          middle: item.middle_band || 0,
          lower: item.lower_band || 0,
        }));
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
  console.log("📍 About to call buildMiniCharts - layout mode:", indicatorsStore.layoutMode, "data loaded:", {
    rsi: rsiMiniData.value.length,
    macd: macdMiniData.value.length,
    bollinger: bollingerMiniData.value.length,
  });
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
  console.log("🎯 buildMiniCharts called - Data status:", {
    rsi: rsiMiniData.value.length,
    macd: macdMiniData.value.length,
    bollinger: bollingerMiniData.value.length,
  });
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
  console.log("🔧 buildMACDMiniChart called - ref:", !!macdMiniChartRef.value, "data length:", macdMiniData.value.length);
  if (!macdMiniChartRef.value || macdMiniData.value.length === 0) {
    console.log("❌ MACD mini chart not built - missing ref or data");
    return;
  }

  try {
    macdMiniChart?.destroy();
    console.log("✅ Building MACD mini chart with", macdMiniData.value.length, "points");

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
  console.log("🔧 buildBollingerMiniChart called - ref:", !!bollingerMiniChartRef.value, "data length:", bollingerMiniData.value.length);
  if (!bollingerMiniChartRef.value || bollingerMiniData.value.length === 0) {
    console.log("❌ Bollinger mini chart not built - missing ref or data");
    return;
  }

  try {
    bollingerMiniChart?.destroy();
    console.log("✅ Building Bollinger mini chart with", bollingerMiniData.value.length, "points");

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

// Helper: Convert timeframe interval to milliseconds
function getIntervalInMs(interval: string): number {
  const unit = interval.slice(-1);
  const value = parseInt(interval.slice(0, -1));

  switch (unit) {
    case "m":
      return value * 60 * 1000; // minutes to ms
    case "h":
      return value * 60 * 60 * 1000; // hours to ms
    case "d":
      return value * 24 * 60 * 60 * 1000; // days to ms
    default:
      return 60 * 1000; // default to 1 minute
  }
}

// Progressive loading: Set default zoom level to show DEFAULT_ZOOM_CANDLES
function setDefaultZoom() {
  if (!candles.value.length) return;

  const activeChart =
    chartMode.value === "line" ? lineChartRef.value : candleChartRef.value;
  if (!activeChart?.chart) return;

  const chart = activeChart.chart;
  if (!chart.scales?.x) return;

  try {
    // Calculate time range for DEFAULT_ZOOM_CANDLES
    const latestTime = new Date(
      candles.value[candles.value.length - 1].time
    ).getTime();
    const intervalMs = getIntervalInMs(selectedTimeframe.value);
    const minTime = latestTime - DEFAULT_ZOOM_CANDLES * intervalMs;

    // Apply zoom programmatically
    chart.zoomScale("x", { min: minTime, max: latestTime }, "none");
    console.log(
      `🔍 Default zoom set to show ${DEFAULT_ZOOM_CANDLES} candles`
    );
  } catch (error) {
    console.error("Error setting default zoom:", error);
  }
}

// Progressive loading: Load more historical candles when approaching data boundary
async function loadMoreHistoricalData() {
  if (isLoadingMore.value || !hasMoreHistoricalData.value) return;
  if (!loadedTimeRange.value.earliest) return;

  isLoadingMore.value = true;
  console.log("📥 Loading more historical data...");

  try {
    const symbol = symbolPair.value;
    const interval = selectedTimeframe.value;
    const beforeTime = loadedTimeRange.value.earliest.toISOString();

    // Fetch FETCH_BATCH_SIZE candles before the earliest loaded candle
    const newCandles = await fetchCandles(
      symbol,
      interval,
      FETCH_BATCH_SIZE,
      undefined, // startTime
      beforeTime // endTime (before earliest)
    );

    if (!newCandles || newCandles.length === 0) {
      console.log("📭 No more historical data available");
      hasMoreHistoricalData.value = false;
      return;
    }

    // Update loaded time range
    loadedTimeRange.value.earliest = new Date(newCandles[0].time);
    hasMoreHistoricalData.value = newCandles.length === FETCH_BATCH_SIZE;

    // Prepend new candles to existing data
    candles.value = [...newCandles, ...candles.value];

    console.log(
      `📥 Loaded ${newCandles.length} more candles. New range: ${loadedTimeRange.value.earliest?.toISOString()} to ${loadedTimeRange.value.latest?.toISOString()}`
    );
  } catch (error) {
    console.error("Error loading more historical data:", error);
  } finally {
    isLoadingMore.value = false;
  }
}

// Progressive loading: Handle boundary detection from chart pan/zoom
async function handleChartBoundary(visibleRange: {
  minVisible: number;
  maxVisible: number;
}) {
  if (isLoadingMore.value || !hasMoreHistoricalData.value) return;
  if (!loadedTimeRange.value.earliest || !loadedTimeRange.value.latest) return;

  const loadedMin = loadedTimeRange.value.earliest.getTime();
  const loadedMax = loadedTimeRange.value.latest.getTime();
  const visibleMin = visibleRange.minVisible;

  // Calculate how far from the data edge the user is viewing (as percentage)
  const dataRange = loadedMax - loadedMin;
  const distanceFromEdge = visibleMin - loadedMin;
  const percentFromEdge = (distanceFromEdge / dataRange) * 100;

  console.log(
    `📊 Boundary check: ${percentFromEdge.toFixed(1)}% from edge (threshold: ${AUTOFETCH_THRESHOLD}%)`
  );

  // If user is within AUTOFETCH_THRESHOLD% of the data edge, load more
  if (percentFromEdge < AUTOFETCH_THRESHOLD) {
    console.log("🚨 Near data boundary! Triggering auto-fetch...");
    await loadMoreHistoricalData();
  }
}

// Initialize the real-time candle builder for 1m timeframe
function initializeCandleBuilder() {
  if (selectedTimeframe.value !== "1m") return;

  candleBuilder.value = new RealtimeCandleBuilder();
  lastProcessedTradeIndex = 0;

  // When a candle is completed, add it to the candles array
  candleBuilder.value.onCandleCompleted((completedCandle, timestamp) => {
    const candleTime = new Date(timestamp);
    console.log(
      `✅ Candle finalisée: ${candleTime.toLocaleTimeString("fr-FR")} - OHLC: ${completedCandle.open.toFixed(2)}/${completedCandle.high.toFixed(2)}/${completedCandle.low.toFixed(2)}/${completedCandle.close.toFixed(2)}`
    );

    // Créer un objet CandleDTO compatible
    const newCandle: CandleDTO = {
      time: candleTime.toISOString(),
      open: completedCandle.open,
      high: completedCandle.high,
      low: completedCandle.low,
      close: completedCandle.close,
      volume: completedCandle.volume,
    };

    // Ajouter à l'array des candles (force Vue reactivity)
    candles.value = [...candles.value, newCandle];
    console.log(`Total candles: ${candles.value.length}`);
    currentBuildingCandle.value = null;
  });

  // When a candle is being built, update the current building candle
  candleBuilder.value.onCandleUpdated((updatingCandle) => {
    // Force Vue reactivity by creating a new object
    currentBuildingCandle.value = {
      open: updatingCandle.open,
      high: updatingCandle.high,
      low: updatingCandle.low,
      close: updatingCandle.close,
      volume: updatingCandle.volume,
    };
  });

  // Process existing trades to rebuild current minute candle
  if (liveTrades.value.length > 0) {
    liveTrades.value.forEach((trade) => {
      candleBuilder.value?.addTrade(trade.price, 0, trade.timestamp);
    });
  }

  // Start polling for new trades every 100ms
  if (tradeProcessingInterval) {
    clearInterval(tradeProcessingInterval);
  }

  tradeProcessingInterval = setInterval(() => {
    if (!candleBuilder.value || selectedTimeframe.value !== "1m") {
      return;
    }

    const newTrades = liveTrades.value.slice(lastProcessedTradeIndex);
    if (newTrades.length > 0) {
      console.log(
        `📊 ${newTrades.length} nouveau(x) trade(s) trouvé(s) via polling`
      );
      newTrades.forEach((trade) => {
        candleBuilder.value?.addTrade(trade.price, 0, trade.timestamp);
      });
      lastProcessedTradeIndex = liveTrades.value.length;
    } else {
      // Même sans nouveaux trades, force le re-render de la candle en construction
      // pour que les petites modifications s'affichent
      if (currentBuildingCandle.value) {
        currentBuildingCandle.value = {
          ...currentBuildingCandle.value,
        };
      }
    }
  }, 100);
} // Process incoming trades through the candle builder via polling in initializeCandleBuilder
// (removed watch-based approach in favor of setInterval polling)

watch(latestCandle, (newCandle) => {
  if (newCandle && candles.value.length > 0) {
    const lastIndex = candles.value.length - 1;
    candles.value[lastIndex] = newCandle;
  }
});

watch(
  () => indicatorsStore.selectedTimeframe,
  async (newTimeframe) => {
    // Toujours charger les données pour cette timeframe
    await loadData();

    // Si c'est 1m, réinitialiser le builder pour qu'il refonctionne
    if (newTimeframe === "1m") {
      console.log("Reinitializing builder after loadData");
      initializeCandleBuilder();
    } else {
      // Arrêter le builder pour les autres timeframes
      if (tradeProcessingInterval) {
        clearInterval(tradeProcessingInterval);
        tradeProcessingInterval = null;
      }
      candleBuilder.value = null;
      currentBuildingCandle.value = null;
      lastProcessedTradeIndex = 0;
    }
  }
);

// Watch layout mode changes to rebuild mini charts when switching to compact mode
watch(
  () => indicatorsStore.layoutMode,
  async (newMode) => {
    console.log("Layout mode changed to:", newMode);
    if (newMode === "compact") {
      // Wait for DOM to update and rebuild mini charts
      await nextTick();
      setTimeout(() => {
        console.log("Building mini charts after layout mode change...");
        buildMiniCharts();
      }, 200);
    }
  }
);

onMounted(async () => {
  console.log("🔧 TradingChart mounted, initializing...");
  await loadData();

  // Initialize real-time candle builder for 1m timeframe
  if (selectedTimeframe.value === "1m") {
    console.log("🟢 Initializing candle builder for 1m");
    initializeCandleBuilder();
  } else {
    console.log(
      "🔴 Not initializing builder - timeframe is:",
      selectedTimeframe.value
    );
  }

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
  if (tradeProcessingInterval) {
    clearInterval(tradeProcessingInterval);
  }
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
            :building-candle-point="
              selectedTimeframe === '1m' && currentBuildingCandle
                ? {
                    x: Date.now(),
                    y: currentBuildingCandle.close,
                  }
                : null
            "
            ref="lineChartRef"
            @pan-complete="handleChartBoundary"
            @zoom-complete="handleChartBoundary"
          />
          <CandleChart
            v-else-if="chartMode === 'candle'"
            :candles="candles"
            :timeframe="selectedTimeframe"
            :building-candle="buildingCandleForChart"
            ref="candleChartRef"
            @pan-complete="handleChartBoundary"
            @zoom-complete="handleChartBoundary"
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
