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

// Charger les données mock selon le timeframe
async function loadMockData() {
  try {
    // Utiliser le timeframe du store
    const timeframe = indicatorsStore.selectedTimeframe;

    if (import.meta.env.VITE_USE_MOCK === "true") {
      console.log(`Loading MACD mock data for timeframe: ${timeframe}`);

      // Importer les données unifiées (mêmes que TradingChart)
      const { default: unifiedData } = await import(
        "@/services/mocks/candles_unified.json"
      );

      let candleData = [];

      // Sélectionner les données selon le timeframe (même logique que TradingChart)
      switch (timeframe) {
        case "1h":
          const oneDayData = unifiedData["1d"] || [];
          candleData = oneDayData.slice(-60); // Dernière heure en minutes
          break;
        case "1d":
          const twentyFourHourData = unifiedData["1d"] || [];
          candleData = twentyFourHourData.slice(-1440); // Dernières 24 heures (1440 minutes)
          break;
        case "7d":
          candleData = unifiedData["7d"] || []; // 168 points (heures)
          break;
        case "1M":
          candleData = unifiedData["1M"] || []; // 180 points (4h)
          break;
        case "1y":
          candleData = unifiedData["1y"] || []; // 365 points (jours)
          break;
        case "all":
          candleData = unifiedData["all"] || []; // 520 points (semaines)
          break;
        default:
          const fallbackData = unifiedData["1d"] || [];
          candleData = fallbackData.slice(-60);
      }

      // Calculer des valeurs MACD réalistes basées sur les prix de clôture
      macdData.value = calculateMACDFromCandles(candleData);

      console.log(
        `Loaded ${macdData.value.length} MACD data points for ${timeframe}`
      );
    } else {
      // Pour l'API réelle, adapter selon le timeframe
      console.error("API MACD data loading not implemented yet");
      // Fallback avec quelques données par défaut
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

// Calculer MACD à partir des données de bougies
function calculateMACDFromCandles(candles: any[]): Array<{
  timestamp: number;
  macd: number;
  signal: number;
  histogram: number;
}> {
  if (candles.length < 26) {
    // Pas assez de données pour calculer MACD, retourner des valeurs par défaut
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

  // Calculer les EMA pour MACD
  const prices = candles.map((c) => c.c);
  const fastEMA = calculateEMA(prices, fastPeriod);
  const slowEMA = calculateEMA(prices, slowPeriod);

  // Calculer la ligne MACD
  const macdLine = [];
  for (let i = slowPeriod - 1; i < candles.length; i++) {
    macdLine.push(fastEMA[i] - slowEMA[i]);
  }

  // Calculer la ligne de signal (EMA du MACD)
  const signalLine = calculateEMA(macdLine, signalPeriod);

  // Construire le résultat final
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
      timestamp: candles[i].t,
      macd: macd,
      signal: signal,
      histogram: histogram,
    });
  }

  return result;
}

// Fonction utilitaire pour calculer l'EMA
function calculateEMA(prices: number[], period: number): number[] {
  const ema = [];
  const multiplier = 2 / (period + 1);

  // Premier point = SMA
  let sum = 0;
  for (let i = 0; i < period && i < prices.length; i++) {
    sum += prices[i];
  }
  ema.push(sum / Math.min(period, prices.length));

  // Points suivants = EMA
  for (let i = period; i < prices.length; i++) {
    const currentEMA: number =
      prices[i] * multiplier + ema[ema.length - 1] * (1 - multiplier);
    ema.push(currentEMA);
  }

  return ema;
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

// Recharger les données quand le timeframe change
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
