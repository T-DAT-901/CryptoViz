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

// Données MACD à partir du fichier mock
const macdData = ref<
  Array<{ timestamp: number; macd: number; signal: number; histogram: number }>
>([]);

// Charger les données mock
async function loadMockData() {
  try {
    const { default: mockData } = await import(
      "@/services/mocks/ind_macd.json"
    );

    // Transformer les données mock au format attendu
    macdData.value = mockData.map((item: any) => ({
      timestamp: item.ts,
      macd: item.macd,
      signal: item.signal,
      histogram: item.hist,
    }));

    console.log(`Loaded ${macdData.value.length} MACD data points`);
  } catch (error) {
    console.error("Error loading MACD mock data:", error);
    // Fallback avec quelques données par défaut
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

// Données pour Chart.js
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
    type: "line", // Type par défaut, les datasets overrides
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

// Reconstruire le chart quand les données changent
watch(() => macdData.value, buildChart, { deep: true });
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
