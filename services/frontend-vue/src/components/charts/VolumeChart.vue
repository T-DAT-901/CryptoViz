<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from "vue";
import {
  Chart,
  BarController,
  BarElement,
  LinearScale,
  TimeScale,
  Tooltip,
  type ChartOptions,
} from "chart.js";
import "chartjs-adapter-date-fns";

Chart.register(BarController, BarElement, LinearScale, TimeScale, Tooltip);

interface VolumeDataPoint {
  timestamp: number;
  volume: number;
  color: string; // 'green' if bullish (close > open), 'red' if bearish
}

const props = defineProps<{
  volumes: VolumeDataPoint[];
  timeframe: string;
}>();

const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;

const chartData = computed(() => {
  return {
    datasets: [
      {
        label: "Volume",
        data: props.volumes.map((v) => ({
          x: v.timestamp,
          y: v.volume,
        })),
        backgroundColor: props.volumes.map((v) =>
          v.color === "green"
            ? "rgba(16, 185, 129, 0.6)"
            : "rgba(239, 68, 68, 0.6)"
        ),
        borderColor: props.volumes.map((v) =>
          v.color === "green" ? "#10b981" : "#ef4444"
        ),
        borderWidth: 1,
        barThickness: "flex",
        maxBarThickness: 8,
      },
    ],
  };
});

const chartOptions: ChartOptions<"bar"> = {
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
          const value = context.parsed.y;
          return `Volume: ${value.toLocaleString("fr-FR")}`;
        },
      },
    },
  },
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
    y: {
      position: "right",
      grid: {
        color: "rgba(255,255,255,0.08)",
      },
      ticks: {
        color: "rgba(255,255,255,0.7)",
        font: { size: 10 },
        callback: function (value) {
          // Format large numbers (K, M, B)
          const num = Number(value);
          if (num >= 1e9) return (num / 1e9).toFixed(1) + "B";
          if (num >= 1e6) return (num / 1e6).toFixed(1) + "M";
          if (num >= 1e3) return (num / 1e3).toFixed(1) + "K";
          return num.toFixed(0);
        },
      },
    },
  },
};

function buildChart() {
  if (!canvasEl.value || !props.volumes.length) return;

  if (chart) {
    chart.destroy();
  }

  const ctx = canvasEl.value.getContext("2d");
  if (!ctx) return;

  chart = new Chart(ctx, {
    type: "bar",
    data: chartData.value,
    options: chartOptions,
  });
}

onMounted(() => {
  buildChart();
});

watch(
  () => props.volumes,
  () => {
    buildChart();
  },
  { deep: true }
);

onBeforeUnmount(() => {
  if (chart) {
    chart.destroy();
  }
});
</script>

<template>
  <div class="volume-chart">
    <canvas ref="canvasEl"></canvas>
  </div>
</template>

<style scoped>
.volume-chart {
  width: 100%;
  height: 100%;
  min-height: 80px;
}

.volume-chart canvas {
  width: 100% !important;
  height: 100% !important;
}
</style>
