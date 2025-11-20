<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, computed } from "vue";
import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Filler,
  type ChartData,
  type ChartOptions,
} from "chart.js";
import "chartjs-adapter-date-fns";
import { useIndicatorsStore } from "@/stores/indicators";
import { transformOldCandlesArray } from "@/utils/mockTransform";

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Filler
);

defineProps<{ symbol: string }>();
const indicatorsStore = useIndicatorsStore();
const canvasEl = ref<HTMLCanvasElement | null>(null);
const bollingerMiniChartRef = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;
let miniChart: Chart | null = null;

// Bollinger Bands data placeholder
const bollingerData = ref<
  Array<{
    timestamp: number;
    upper: number;
    middle: number;
    lower: number;
  }>
>([]);

function calculateBollingerBands(candles: any[], period: number = 20) {
  const result = [];
  for (let i = period; i < candles.length; i++) {
    // Take the last 20 candles
    const slice = candles.slice(i - period, i);

    // Calculate SMA
    const price = slice.map((candle) => candle.close);
    const sma = price.reduce((sum, price) => sum + price, 0) / period;

    // Calculate standard deviation
    const variance =
      price.reduce((sum, price) => sum + Math.pow(price - sma, 2), 0) / period;
    const stdDev = Math.sqrt(variance);

    // Calculate bands
    result.push({
      timestamp: new Date(candles[i].time).getTime(),
      upper: sma + 2 * stdDev,
      middle: sma,
      lower: sma - 2 * stdDev,
    });
  }

  return result;
}

async function loadData() {
  try {
    const timeframe = indicatorsStore.selectedTimeframe;

    if (import.meta.env.VITE_USE_MOCK === "true") {
      const { default: unifiedData } = await import(
        "@/services/mocks/candles_unified.json"
      );

      let candleData = [];
      switch (timeframe) {
        case "1d":
          candleData = transformOldCandlesArray(unifiedData["1d"] || []);
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
          candleData = transformOldCandlesArray(unifiedData["1d"] || []);
      }

      // Calculate Bollinger Bands from price data
      bollingerData.value = calculateBollingerBands(candleData, 20);
    }
  } catch (error) {
    console.error("Error loading Bollinger Bands data:", error);
    bollingerData.value = [];
  }
}

// Chart data with 3 bands and fill between them
const chartData = computed((): ChartData<"line"> => {
  const data = bollingerData.value;

  return {
    datasets: [
      {
        label: "Upper Band",
        data: data.map((point) => ({
          x: point.timestamp,
          y: point.upper,
        })),
        borderColor: "rgba(239, 68, 68, 0.7)",
        backgroundColor: "transparent",
        borderWidth: 1,
        pointRadius: 0,
        fill: false,
      },
      {
        label: "Middle Band (SMA)",
        data: data.map((point) => ({
          x: point.timestamp,
          y: point.middle,
        })),
        borderColor: "rgb(59, 130, 246)",
        backgroundColor: "transparent",
        borderWidth: 2,
        pointRadius: 0,
        fill: false,
      },
      {
        label: "Lower Band",
        data: data.map((point) => ({
          x: point.timestamp,
          y: point.lower,
        })),
        borderColor: "rgba(239, 68, 68, 0.7)",
        backgroundColor: "rgba(59, 130, 246, 0.1)",
        borderWidth: 1,
        pointRadius: 0,
        fill: "+1",
      },
    ],
  };
});

// Chart configuration
const chartOptions = computed(
  (): ChartOptions<"line"> => ({
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      intersect: false,
      mode: "index",
    },
    plugins: {
      legend: {
        display: true,
        position: "top",
        labels: {
          color: "rgb(156, 163, 175)",
          font: {
            size: 12,
          },
        },
      },
      tooltip: {
        backgroundColor: "rgba(17, 24, 39, 0.95)",
        titleColor: "rgb(229, 231, 235)",
        bodyColor: "rgb(229, 231, 235)",
        borderColor: "rgba(75, 85, 99, 0.5)",
        borderWidth: 1,
        cornerRadius: 8,
        displayColors: false,
        callbacks: {
          title: function (context) {
            if (context[0]) {
              const date = new Date(context[0].parsed.x);
              return date.toLocaleDateString("fr-FR", {
                day: "2-digit",
                month: "2-digit",
                hour: "2-digit",
                minute: "2-digit",
              });
            }
            return "";
          },
          label: function (context) {
            const value = context.parsed.y;
            return `${context.dataset.label}: ${value.toFixed(2)}`;
          },
        },
      },
    },
    scales: {
      x: {
        type: "time",
        time: {
          displayFormats: {
            minute: "HH:mm",
            hour: "HH:mm",
            day: "dd/MM",
          },
        },
        grid: {
          color: "rgba(75, 85, 99, 0.2)",
        },
        ticks: {
          color: "rgb(156, 163, 175)",
          maxTicksLimit: 8,
        },
      },
      y: {
        grid: {
          color: "rgba(75, 85, 99, 0.2)",
        },
        ticks: {
          color: "rgb(156, 163, 175)",
        },
      },
    },
  })
);

// Setup and render chart
function buildChart() {
  if (!canvasEl.value) return;

  if (chart) {
    chart.destroy();
  }

  const ctx = canvasEl.value.getContext("2d");
  if (!ctx) return;

  chart = new Chart(ctx, {
    type: "line",
    data: chartData.value,
    options: chartOptions.value,
  });
}

// Mini chart for compact view
const miniChartOptions = computed(
  (): ChartOptions<"line"> => ({
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        enabled: false,
      },
    },
    scales: {
      x: {
        display: false,
      },
      y: {
        display: false,
      },
    },
  })
);

function buildMiniChart() {
  if (!bollingerMiniChartRef.value) return;

  if (miniChart) {
    miniChart.destroy();
  }

  const ctx = bollingerMiniChartRef.value.getContext("2d");
  if (!ctx) return;

  miniChart = new Chart(ctx, {
    type: "line",
    data: chartData.value,
    options: miniChartOptions.value,
  });
}

// Initialize chart and watch for updates
onMounted(() => {
  loadData().then(() => {
    buildChart();
    buildMiniChart();
  });
});

watch(
  () => indicatorsStore.selectedTimeframe,
  () => {
    loadData().then(() => {
      buildChart();
      buildMiniChart();
    });
  }
);

watch(
  () => bollingerData.value,
  () => {
    buildChart();
    buildMiniChart();
  },
  { deep: true }
);

onBeforeUnmount(() => {
  if (chart) {
    chart.destroy();
  }
  if (miniChart) {
    miniChart.destroy();
  }
});
</script>

<template>
  <div class="bollinger-chart">
    <div class="bollinger-chart__header">
      <h3 class="bollinger-chart__title">Bollinger Bands</h3>
      <span class="bollinger-chart__symbol">{{ symbol }}</span>
    </div>
    <div class="bollinger-chart__content">
      <canvas
        v-if="indicatorsStore.layoutMode === 'detailed'"
        ref="canvasEl"
      ></canvas>
      <canvas
        v-else
        ref="bollingerMiniChartRef"
        class="bollinger-chart-mini-canvas"
      ></canvas>
    </div>
  </div>
</template>
