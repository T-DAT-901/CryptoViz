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
import zoomPlugin from "chartjs-plugin-zoom";
import "chartjs-adapter-date-fns";
import { useIndicatorsStore } from "@/stores/indicators";
import { fetchMACD } from "@/services/indicators.api";

Chart.register(
  LineController,
  BarController,
  LineElement,
  BarElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  zoomPlugin
);

const props = defineProps<{ symbol: string }>();
const indicatorsStore = useIndicatorsStore();

const canvasEl = ref<HTMLCanvasElement | null>(null);
const macdMiniChartRef = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;
let miniChart: Chart | null = null;

// MACD data from mock file
const macdData = ref<
  Array<{ timestamp: number; macd: number; signal: number; histogram: number }>
>([]);

// Load MACD data from API
async function loadMACDData() {
  try {
    const timeframe = indicatorsStore.selectedTimeframe;
    console.log(`Loading MACD data for: ${props.symbol} (${timeframe})`);
    const rawData = await fetchMACD(
      props.symbol,
      timeframe,
      indicatorsStore.macdFast,
      indicatorsStore.macdSlow,
      indicatorsStore.macdSignal,
      10000
    );

    if (rawData && rawData.length > 0) {
      // Transform backend format to chart format
      macdData.value = rawData.map((item: any) => ({
        timestamp: new Date(item.time).getTime(),
        macd: item.value || 0,
        signal: item.value_signal || 0,
        histogram: item.value_histogram || 0,
      }));
      console.log(`Loaded ${macdData.value.length} MACD data points`);
    } else {
      console.warn("No MACD data received from API");
      macdData.value = [];
    }
  } catch (error) {
    console.error("Error loading MACD data:", error);
    macdData.value = [];
  }
}

// Calculate MACD from candle data
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
      position: "bottom",
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
          day: "dd/MM",
        },
      },
    },
    x2: {
      type: "time",
      position: "bottom",
      offset: true,
      grid: {
        display: false,
      },
      ticks: {
        color: "rgba(255,255,255,0.5)",
        font: { size: 9 },
        maxTicksLimit: 5,
      },
      time: {
        displayFormats: {
          minute: "dd/MM",
          hour: "dd/MM",
          day: "dd/MM/yyyy",
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
    zoom: {
      pan: {
        enabled: true,
        mode: "x",
        modifierKey: undefined,
      },
      zoom: {
        wheel: {
          enabled: true,
          modifierKey: "ctrl",
          speed: 0.05,
        },
        pinch: {
          enabled: true,
        },
        mode: "x",
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

// Mini chart for compact view
const miniChartOptions: ChartOptions = {
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
};

function buildMiniChart() {
  if (!macdMiniChartRef.value) return;

  if (miniChart) {
    miniChart.destroy();
  }

  const ctx = macdMiniChartRef.value.getContext("2d");
  if (!ctx) return;

  miniChart = new Chart(ctx, {
    type: "line",
    data: chartData.value,
    options: miniChartOptions,
  });
}

onMounted(async () => {
  await loadMACDData();
  buildChart();
  buildMiniChart();
});

onBeforeUnmount(() => {
  chart?.destroy();
  miniChart?.destroy();
});

// Rebuild chart when data changes
watch(() => macdData.value, () => {
  buildChart();
  buildMiniChart();
}, { deep: true });

// Reload data when timeframe changes
watch(
  () => indicatorsStore.selectedTimeframe,
  async () => {
    console.log(
      "MACD Chart: Timeframe changed to",
      indicatorsStore.selectedTimeframe
    );
    await loadMACDData();
    buildChart();
    buildMiniChart();
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
      <canvas
        v-if="indicatorsStore.layoutMode === 'detailed'"
        ref="canvasEl"
      ></canvas>
      <canvas
        v-else
        ref="macdMiniChartRef"
        class="macd-chart-mini-canvas"
      ></canvas>
    </div>
  </div>
</template>
