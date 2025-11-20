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

const props = defineProps<{ symbol: string }>();
const indicatorsStore = useIndicatorsStore();

const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;

// RSI data from mock file
const rsiData = ref<Array<{ timestamp: number; value: number }>>([]);

// Load mock data based on timeframe
async function loadMockData() {
  try {
    // Utiliser le timeframe du store
    const timeframe = indicatorsStore.selectedTimeframe;

    if (import.meta.env.VITE_USE_MOCK === "true") {
      console.log(`Loading RSI mock data for timeframe: ${timeframe}`);

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

      // Calculer des valeurs RSI réalistes basées sur les prix de clôture
      rsiData.value = calculateRSIFromCandles(candleData);

      console.log(
        `Loaded ${rsiData.value.length} RSI data points for ${timeframe}`
      );
    } else {
      // Pour l'API réelle, adapter selon le timeframe
      console.error("API RSI data loading not implemented yet");
      // Fallback avec quelques données par défaut
      const now = Date.now();
      rsiData.value = Array.from({ length: 20 }, (_, i) => ({
        timestamp: now - (20 - i) * 15 * 60 * 1000,
        value: 30 + Math.sin(i * 0.3) * 20 + Math.random() * 10,
      }));
    }
  } catch (error) {
    console.error("Error loading RSI mock data:", error);
    // Fallback avec quelques données par défaut
    const now = Date.now();
    rsiData.value = Array.from({ length: 20 }, (_, i) => ({
      timestamp: now - (20 - i) * 15 * 60 * 1000,
      value: 30 + Math.sin(i * 0.3) * 20 + Math.random() * 10,
    }));
  }
}

// Calculate RSI from candle data
function calculateRSIFromCandles(
  candles: any[]
): Array<{ timestamp: number; value: number }> {
  if (candles.length < 14) {
    // Not enough data to calculate RSI, return default values
    return candles.map((candle, i) => ({
      timestamp: candle.time,
      value: 50 + Math.sin(i * 0.2) * 20 + Math.random() * 10,
    }));
  }

  const rsiPeriod = indicatorsStore.rsiPeriod;
  const result: Array<{ timestamp: number; value: number }> = [];

  for (let i = 0; i < candles.length; i++) {
    if (i < rsiPeriod - 1) {
      // Les premières valeurs avant qu'on ait assez de données
      result.push({
        timestamp: new Date(candles[i].time).getTime(),
        value: 50 + Math.sin(i * 0.2) * 15,
      });
      continue;
    }

    // Calculer les gains et pertes sur la période
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

    // Calculer RSI
    let rsi = 50; // Valeur par défaut
    if (avgLoss !== 0) {
      const rs = avgGain / avgLoss;
      rsi = 100 - 100 / (1 + rs);
    }

    // S'assurer que RSI est dans la plage [0, 100]
    rsi = Math.max(0, Math.min(100, rsi));

    result.push({
      timestamp: new Date(candles[i].time).getTime(),
      value: rsi,
    });
  }

  return result;
}

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
      display: false, // Pas de légende, on a les labels sur l'axe Y
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
          return ""; // Retourne string vide pour les autres datasets
        },
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
  await loadMockData();
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
    await loadMockData();
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
