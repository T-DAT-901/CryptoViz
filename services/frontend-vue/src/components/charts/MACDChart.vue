<!-- Graph MACD -->
<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, computed } from "vue";
import {
  Chart,
  LineController,
  BarController,
  LineElement,
  BarElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  type ChartData,
  type ChartOptions,
} from "chart.js";
import "chartjs-adapter-date-fns";
import { useIndicatorsStore } from "@/stores/indicators";
import { transformOldCandlesArray } from "@/utils/mockTransform";

Chart.register(
  LineController,
  BarController,
  LineElement,
  BarElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip
);

const props = defineProps<{ symbol: string }>();
const indicatorsStore = useIndicatorsStore();

const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;

// MACD data from mock file
const macdData = ref<
  Array<{ timestamp: number; macd: number; signal: number; histogram: number }>
>([]);

// Load mock data based on timeframe
async function loadMockData() {
  try {
    // Use timeframe from store
    const timeframe = indicatorsStore.selectedTimeframe;

    if (import.meta.env.VITE_USE_MOCK === "true") {
      console.log(`Loading MACD mock data for timeframe: ${timeframe}`);

      // Import unified data (same as TradingChart)
      const { default: unifiedData } = await import(
        "@/services/mocks/candles_unified.json"
      );

      let candleData = [];

      // Select data based on timeframe (same logic as TradingChart)
      switch (timeframe) {
        case "1h":
          const oneDayData = unifiedData["1d"] || [];
          candleData = transformOldCandlesArray(oneDayData.slice(-60));
          break;
        case "1d":
          const twentyFourHourData = unifiedData["1d"] || [];
          candleData = transformOldCandlesArray(
            twentyFourHourData.slice(-1440)
          );
          break;
        case "7d":
          candleData = transformOldCandlesArray(unifiedData["7d"] || []);
          break;
        case "1M":
          candleData = transformOldCandlesArray(unifiedData["1M"] || []);
          break;
        case "1y":
          candleData = transformOldCandlesArray(unifiedData["1y"] || []);
          break;
        case "all":
          candleData = transformOldCandlesArray(unifiedData["all"] || []);
          break;
        default:
          const fallbackData = unifiedData["1d"] || [];
          candleData = transformOldCandlesArray(fallbackData.slice(-60));
      }

      // Calculate realistic MACD values based on closing prices
      macdData.value = calculateMACDFromCandles(candleData);

      console.log(
        `Loaded ${macdData.value.length} MACD data points for ${timeframe}`
      );
    } else {
      // For real API, adapt based on timeframe
      console.error("API MACD data loading not implemented yet");
      // Fallback with some default data
      macdData.value = [
        {
          timestamp: Date.now() - 120000,
          macd: 0.5,
          signal: 0.4,
          histogram: 0.1,
        },
        {
          timestamp: Date.now() - 60000,
          macd: 0.8,
          signal: 0.6,
          histogram: 0.2,
        },
        { timestamp: Date.now(), macd: 0.7, signal: 0.65, histogram: 0.05 },
      ];
    }
  } catch (error) {
    console.error("Error loading MACD mock data:", error);
    // Fallback with some default data
    macdData.value = [
      {
        timestamp: Date.now() - 120000,
        macd: 0.5,
        signal: 0.4,
        histogram: 0.1,
      },
      { timestamp: Date.now() - 60000, macd: 0.8, signal: 0.6, histogram: 0.2 },
      { timestamp: Date.now(), macd: 0.7, signal: 0.65, histogram: 0.05 },
    ];
  }
}

// Calculate MACD from candle data
function calculateMACDFromCandles(candles: any[]): Array<{
  timestamp: number;
  macd: number;
  signal: number;
  histogram: number;
}> {
  if (candles.length < 26) {
    // Not enough data to calculate MACD, return default values
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

  // Calculate EMAs for MACD
  const prices = candles.map((c) => c.close);
  const fastEMA = calculateEMA(prices, fastPeriod);
  const slowEMA = calculateEMA(prices, slowPeriod);

  // Calculate MACD line
  const macdLine = [];
  for (let i = slowPeriod - 1; i < candles.length; i++) {
    macdLine.push(fastEMA[i] - slowEMA[i]);
  }

  // Calculate signal line (EMA of MACD)
  const signalLine = calculateEMA(macdLine, signalPeriod);

  // Build final result
  for (let i = 0; i < candles.length; i++) {
    let macd = 0;
    let signal = 0;
    let histogram = 0;

    if (i >= slowPeriod - 1) {
      const macdIndex = i - (slowPeriod - 1);
      macd = macdLine[macdIndex] || 0;

      if (macdIndex >= signalPeriod - 1) {
        const signalIndex = macdIndex - (signalPeriod - 1);
        signal = signalLine[signalIndex] || 0;
        histogram = macd - signal;
      }
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

// Utility function to calculate EMA
function calculateEMA(prices: number[], period: number): number[] {
  const ema = [];
  const multiplier = 2 / (period + 1);

  // First point = SMA
  let sum = 0;
  for (let i = 0; i < period && i < prices.length; i++) {
    sum += prices[i];
  }
  ema.push(sum / Math.min(period, prices.length));

  // Following points = EMA
  for (let i = period; i < prices.length; i++) {
    const currentEMA: number =
      prices[i] * multiplier + ema[ema.length - 1] * (1 - multiplier);
    ema.push(currentEMA);
  }

  return ema;
}

// Data for Chart.js
const chartData = computed(() => {
  const timestamps = macdData.value.map((d) => d.timestamp);

  return {
    labels: timestamps,
    datasets: [
      {
        type: "line" as const,
        label: "MACD",
        data: macdData.value.map((d) => ({ x: d.timestamp, y: d.macd })),
        borderColor: "#3b82f6", // Bleu
        backgroundColor: "transparent",
        borderWidth: 2,
        pointRadius: 0,
        pointHoverRadius: 4,
        tension: 0.1,
        yAxisID: "y",
      },
      {
        type: "line" as const,
        label: "Signal",
        data: macdData.value.map((d) => ({ x: d.timestamp, y: d.signal })),
        borderColor: "#f59e0b", // Orange
        backgroundColor: "transparent",
        borderWidth: 2,
        pointRadius: 0,
        pointHoverRadius: 4,
        tension: 0.1,
        yAxisID: "y",
      },
      {
        type: "bar" as const,
        label: "Histogramme",
        data: macdData.value.map((d) => ({ x: d.timestamp, y: d.histogram })),
        backgroundColor: macdData.value.map((d) =>
          d.histogram >= 0
            ? "rgba(16, 185, 129, 0.6)"
            : "rgba(239, 68, 68, 0.6)"
        ),
        borderColor: macdData.value.map((d) =>
          d.histogram >= 0 ? "#10b981" : "#ef4444"
        ),
        borderWidth: 1,
        yAxisID: "y",
        barThickness: 3,
      },
    ],
  };
});

const options: ChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: "index" },
  scales: {
    x: {
      type: "time",
      grid: {
        color: "rgba(255,255,255,0.08)",
      },
      ticks: {
        color: "rgba(255,255,255,0.7)",
        font: { size: 10 },
        maxTicksLimit: 6,
      },
      time: {
        displayFormats: {
          minute: "HH:mm",
          hour: "HH:mm",
        },
      },
    },
    y: {
      position: "right",
      grid: {
        color: "rgba(255,255,255,0.08)",
      },
      ticks: {
        color: "rgba(255,255,255,0.7)",
        font: { size: 10 },
        callback: function (value) {
          return Number(value).toFixed(3);
        },
      },
    },
  },
  plugins: {
    legend: {
      display: true,
      position: "top",
      labels: {
        color: "rgba(255,255,255,0.8)",
        font: { size: 11 },
        usePointStyle: true,
        pointStyle: "line",
      },
    },
    tooltip: {
      backgroundColor: "rgba(7, 14, 16, 0.95)",
      titleColor: "#fff",
      bodyColor: "#fff",
      borderColor: "rgba(255,255,255,0.2)",
      borderWidth: 1,
      callbacks: {
        title: function (context) {
          return new Date(context[0].parsed.x).toLocaleString("fr-FR");
        },
        label: function (context) {
          const label = context.dataset.label || "";
          const value = Number(context.parsed.y).toFixed(4);
          return `${label}: ${value}`;
        },
      },
    },
  },
};

function buildChart() {
  if (!canvasEl.value || !macdData.value.length) return;

  chart?.destroy();
  chart = new Chart(canvasEl.value, {
    type: "line", // Default type, datasets override
    data: chartData.value,
    options,
  });
}

onMounted(async () => {
  await loadMockData();
  buildChart();
});

onBeforeUnmount(() => {
  chart?.destroy();
});

// Rebuild chart when data changes
watch(() => macdData.value, buildChart, { deep: true });

// Reload data when timeframe changes
watch(
  () => indicatorsStore.selectedTimeframe,
  async () => {
    console.log(
      "MACD Chart: Timeframe changed to",
      indicatorsStore.selectedTimeframe
    );
    await loadMockData();
    buildChart();
  }
);
</script>

<template>
  <div class="macd-chart">
    <div class="macd-chart__header">
      <h3 class="macd-chart__title">
        MACD ({{ indicatorsStore.macdFast }},{{ indicatorsStore.macdSlow }},{{
          indicatorsStore.macdSignal
        }})
      </h3>
      <span class="macd-chart__symbol">{{ symbol }}</span>
    </div>
    <div class="macd-chart__content">
      <canvas ref="canvasEl"></canvas>
    </div>
  </div>
</template>
