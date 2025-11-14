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

const momentumData = ref<Array<{ timestamp: number; value: number }>>([]);

// Calculate momentum as percentage change over 14 periods
function calculateMomentum(candles: any[], period: number = 14) {
  const momentum = [];

  for (let i = period; i < candles.length; i++) {
    const currentPrice = candles[i].c;
    const pastPrice = candles[i - period].c;
    const momentumValue = ((currentPrice - pastPrice) / pastPrice) * 100;

    momentum.push({
      timestamp: candles[i].t,
      value: momentumValue,
    });
  }

  return momentum;
}

// Load candle data and calculate momentum
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
          candleData = unifiedData["1d"] || [];
          break;
        case "7d":
          candleData = unifiedData["7d"] || [];
          break;
        case "1M":
          candleData = unifiedData["1M"] || [];
          break;
        case "1y":
          candleData = unifiedData["1y"] || [];
          break;
        case "all":
          candleData = unifiedData["all"] || [];
          break;
        default:
          candleData = unifiedData["1d"] || [];
      }

      // Calculate momentum from price data
      momentumData.value = calculateMomentum(candleData, 14);
    }
  } catch (error) {
    console.error("Error loading momentum data:", error);
    momentumData.value = [];
  }
}

// Chart data with momentum line and zero reference
const chartData = computed((): ChartData<"line"> => {
  const data = momentumData.value;

  return {
    datasets: [
      {
        label: "Momentum",
        data: data.map((point) => ({
          x: point.timestamp,
          y: point.value,
        })),
        borderColor: "rgb(59, 130, 246)",
        backgroundColor: "rgba(59, 130, 246, 0.1)",
        borderWidth: 2,
        pointRadius: 0,
        pointHoverRadius: 4,
        fill: false,
        tension: 0.1,
      },
      // Zero line for better readability
      {
        label: "Zero Line",
        data: data.map((point) => ({
          x: point.timestamp,
          y: 0,
        })),
        borderColor: "rgba(156, 163, 175, 0.5)",
        backgroundColor: "transparent",
        borderWidth: 1,
        pointRadius: 0,
        pointHoverRadius: 0,
        borderDash: [5, 5],
        fill: false,
      },
    ],
  };
});

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
        display: false,
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
            return `Momentum: ${value.toFixed(2)}%`;
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
          callback: function (value) {
            return value + "%";
          },
        },
      },
    },
  })
);

// Setup chart with proper scaling and styling
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

onMounted(() => {
  loadData().then(() => {
    buildChart();
  });
});

watch(
  () => indicatorsStore.selectedTimeframe,
  () => {
    loadData().then(() => {
      buildChart();
    });
  }
);

watch(() => momentumData.value, buildChart, { deep: true });

onBeforeUnmount(() => {
  if (chart) {
    chart.destroy();
  }
});
</script>

<template>
  <div class="momentum-chart">
    <div class="momentum-chart__header">
      <h3 class="momentum-chart__title">Momentum</h3>
      <span class="momentum-chart__symbol">{{ symbol }}</span>
    </div>
    <div class="momentum-chart__content">
      <canvas ref="canvasEl"></canvas>
    </div>
  </div>
</template>
