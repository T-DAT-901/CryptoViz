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
import zoomPlugin from "chartjs-plugin-zoom";
import "chartjs-adapter-date-fns";
import { useIndicatorsStore } from "@/stores/indicators";
import { fetchIndicators } from "@/services/markets.api";

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Filler,
  zoomPlugin
);

const props = defineProps<{ symbol: string }>();
const indicatorsStore = useIndicatorsStore();

const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;

// RSI data from mock file
const rsiData = ref<Array<{ timestamp: number; value: number }>>([]);

// Load RSI data from API
async function loadRSIData() {
  try {
    console.log(`Loading RSI data for: ${props.symbol}`);
    const data = await fetchIndicators(props.symbol, "rsi");

    if (data && data.length > 0) {
      rsiData.value = data;
      console.log(`Loaded ${rsiData.value.length} RSI data points`);
    } else {
      console.warn("No RSI data received from API");
      rsiData.value = [];
    }
  } catch (error) {
    console.error("Error loading RSI data:", error);
    rsiData.value = [];
  }
}

// Calculate RSI from candle data
// Data for Chart.js
const chartData = computed(() => {
  if (!rsiData.value.length) return { datasets: [] };

  return {
    datasets: [
      // Zone de surachat (70-100)
      {
        label: "Surachat",
        data: rsiData.value.map((d) => ({ x: d.timestamp, y: 100 })),
        backgroundColor: "rgba(239, 68, 68, 0.1)", // Rouge translucide
        borderColor: "transparent",
        fill: "1", // Fill to next dataset
        pointRadius: 0,
        order: 3,
      },
      // Ligne 70
      {
        label: "Niveau 70",
        data: rsiData.value.map((d) => ({ x: d.timestamp, y: 70 })),
        backgroundColor: "rgba(239, 68, 68, 0.2)", // Rouge translucide
        borderColor: "#ef4444",
        borderWidth: 1,
        borderDash: [5, 5],
        pointRadius: 0,
        fill: "2", // Fill to next dataset
        order: 2,
      },
      // Zone neutre (30-70)
      {
        label: "Zone neutre",
        data: rsiData.value.map((d) => ({ x: d.timestamp, y: 30 })),
        backgroundColor: "rgba(148, 163, 184, 0.05)", // Gris très translucide
        borderColor: "#10b981",
        borderWidth: 1,
        borderDash: [5, 5],
        pointRadius: 0,
        fill: "3", // Fill to next dataset
        order: 1,
      },
      // Zone de survente (0-30)
      {
        label: "Survente",
        data: rsiData.value.map((d) => ({ x: d.timestamp, y: 0 })),
        backgroundColor: "rgba(16, 185, 129, 0.1)", // Vert translucide
        borderColor: "transparent",
        pointRadius: 0,
        order: 0,
      },
      // Ligne RSI principale
      {
        label: "RSI",
        data: rsiData.value.map((d) => ({ x: d.timestamp, y: d.value })),
        borderColor: "#3b82f6", // Bleu principal
        backgroundColor: "rgba(59, 130, 246, 0.1)",
        borderWidth: 2,
        pointRadius: 0,
        pointHoverRadius: 5,
        pointBackgroundColor: "#3b82f6",
        pointBorderColor: "#ffffff",
        pointBorderWidth: 2,
        tension: 0.1,
        order: 4, // Au-dessus de tout
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
      min: 0,
      max: 100,
      position: "right",
      grid: {
        color: "rgba(255,255,255,0.08)",
      },
      ticks: {
        color: "rgba(255,255,255,0.7)",
        font: { size: 10 },
        stepSize: 20,
        callback: function (value) {
          // Mettre en évidence les niveaux clés
          if (value === 70) return "70 (Surachat)";
          if (value === 50) return "50 (Neutre)";
          if (value === 30) return "30 (Survente)";
          return value.toString();
        },
      },
      // Lignes horizontales aux niveaux clés
      afterBuildTicks: function (scale) {
        scale.ticks = [
          { value: 0, label: "0" },
          { value: 30, label: "30" },
          { value: 50, label: "50" },
          { value: 70, label: "70" },
          { value: 100, label: "100" },
        ];
      },
    },
  },
  plugins: {
    legend: {
      display: false,
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
          if (context.datasetIndex === 4) {
            // RSI line
            const value = Number(context.parsed.y).toFixed(1);
            let interpretation = "";

            if (context.parsed.y >= 70) {
              interpretation = " (Surachat - Signal de vente)";
            } else if (context.parsed.y <= 30) {
              interpretation = " (Survente - Signal d'achat)";
            } else if (context.parsed.y > 50) {
              interpretation = " (Tendance haussière)";
            } else {
              interpretation = " (Tendance baissière)";
            }

            return `RSI: ${value}${interpretation}`;
          }
          return "";
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
  if (!canvasEl.value || !rsiData.value.length) return;

  chart?.destroy();
  chart = new Chart(canvasEl.value, {
    type: "line",
    data: chartData.value,
    options,
  });
}

onMounted(async () => {
  await loadRSIData();
  buildChart();
});

onBeforeUnmount(() => {
  chart?.destroy();
});

// Rebuild chart when data changes
watch(() => rsiData.value, buildChart, { deep: true });

// Recharger les données quand le timeframe change
watch(
  () => indicatorsStore.selectedTimeframe,
  async () => {
    console.log(
      "RSI Chart: Timeframe changed to",
      indicatorsStore.selectedTimeframe
    );
    await loadRSIData();
    buildChart();
  }
);
</script>

<template>
  <div class="rsi-chart">
    <div class="rsi-chart__header">
      <h3 class="rsi-chart__title">RSI ({{ indicatorsStore.rsiPeriod }})</h3>
      <span class="rsi-chart__symbol">{{ symbol }}</span>
      <!-- Indicateur de signal en temps réel -->
      <div class="rsi-chart__signal" v-if="rsiData.length">
        <span
          class="rsi-chart__signal-badge"
          :class="{
            'rsi-chart__signal-badge--overbought':
              rsiData[rsiData.length - 1]?.value >= 70,
            'rsi-chart__signal-badge--oversold':
              rsiData[rsiData.length - 1]?.value <= 30,
            'rsi-chart__signal-badge--neutral':
              rsiData[rsiData.length - 1]?.value > 30 &&
              rsiData[rsiData.length - 1]?.value < 70,
          }"
        >
          {{
            rsiData[rsiData.length - 1]?.value >= 70
              ? "SURACHAT"
              : rsiData[rsiData.length - 1]?.value <= 30
                ? "SURVENTE"
                : "NEUTRE"
          }}
        </span>
        <span class="rsi-chart__current-value">
          {{ rsiData[rsiData.length - 1]?.value?.toFixed(1) }}
        </span>
      </div>
    </div>
    <div class="rsi-chart__content">
      <canvas ref="canvasEl"></canvas>
    </div>
  </div>
</template>
