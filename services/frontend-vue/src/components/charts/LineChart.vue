<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from "vue";
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

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Filler
);

/** Input point (what the wrapper sends) */
type InPt = { x: number | string | Date; y: number };
const props = defineProps<{ points: InPt[] }>();

const canvasEl = ref<HTMLCanvasElement | null>(null);
const tooltipEl = ref<HTMLDivElement | null>(null);
let chart: Chart<"line"> | null = null;

// État du tooltip personnalisé
const tooltipVisible = ref(false);
const tooltipData = ref({
  x: 0,
  y: 0,
  date: "",
  price: "",
  change: "",
  isPositive: true,
});

/** Normalize x into a numeric timestamp for Chart.js typing */
function toNumericPoints(src: InPt[]) {
  return src.map((p) => ({
    x: typeof p.x === "number" ? p.x : new Date(p.x).getTime(),
    y: p.y,
  }));
}

function build() {
  if (!canvasEl.value) return;

  const ctx = canvasEl.value.getContext("2d")!;
  const grad = ctx.createLinearGradient(0, 0, 0, canvasEl.value.height);
  grad.addColorStop(0, "rgba(16,185,129,0.2)");
  grad.addColorStop(1, "rgba(16,185,129,0)");

  const numericPoints = toNumericPoints(props.points);
  // console.log('LineChart - Points data:', numericPoints.length, numericPoints.slice(0, 3)); // Debug

  const data: ChartData<"line"> = {
    datasets: [
      {
        label: "Prix",
        data: numericPoints,
        parsing: false,
        borderColor: "#10b981",
        backgroundColor: grad,
        borderWidth: 2,
        fill: true,
        tension: 0.1, // Moins arrondi (était 0.3)
        pointRadius: 0,
        pointHoverRadius: 6,
        pointBackgroundColor: "#10b981",
        pointBorderColor: "#ffffff",
        pointBorderWidth: 2,
      },
    ],
  };

  const options: ChartOptions<"line"> = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { intersect: false, mode: "index" },
    onHover: (event, activeElements) => {
      if (activeElements.length > 0) {
        const dataIndex = activeElements[0].index;
        const datasetIndex = activeElements[0].datasetIndex;
        const point = chart?.data.datasets[datasetIndex].data[dataIndex] as any;
        const element = activeElements[0].element;

        if (point && canvasEl.value && element) {
          const rect = canvasEl.value.getBoundingClientRect();

          // Position par rapport au graphique, pas à la souris
          const tooltipX = rect.left + element.x;
          const tooltipY = rect.top + element.y - 100; // 100px au-dessus du point

          // Calculer les infos du tooltip
          const date = new Date(point.x);
          const price = point.y;

          // Variation par rapport au point précédent
          let change = 0;
          let changePercent = 0;
          if (dataIndex > 0) {
            const previousPoint = chart?.data.datasets[datasetIndex].data[
              dataIndex - 1
            ] as any;
            change = price - previousPoint.y;
            changePercent = (change / previousPoint.y) * 100;
          }

          tooltipData.value = {
            x: tooltipX,
            y: tooltipY,
            date:
              date.toLocaleDateString("fr-FR", {
                day: "2-digit",
                month: "2-digit",
                year: "numeric",
              }) +
              " " +
              date.toLocaleTimeString("fr-FR", {
                hour: "2-digit",
                minute: "2-digit",
              }),
            price:
              price.toLocaleString("fr-FR", {
                minimumFractionDigits: 2,
                maximumFractionDigits: 2,
              }) + " €",
            change: `${change > 0 ? "+" : ""}${change.toFixed(2)} € (${change > 0 ? "+" : ""}${changePercent.toFixed(2)}%)`,
            isPositive: change >= 0,
          };

          tooltipVisible.value = true;
        }
      } else {
        tooltipVisible.value = false;
      }
    },
    scales: {
      x: {
        type: "time",
        grid: {
          color: "rgba(255,255,255,0.08)",
        },
        ticks: {
          color: "rgba(255,255,255,0.7)",
          font: { size: 11 },
        },
      },
      y: {
        position: "right",
        grid: {
          color: "rgba(255,255,255,0.08)",
        },
        ticks: {
          color: "rgba(255,255,255,0.7)",
          font: { size: 11 },
          callback: function (value) {
            return Number(value).toLocaleString("fr-FR") + " €";
          },
        },
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: {
        enabled: false, // Désactiver le tooltip par défaut
      },
    },
    elements: {
      point: { radius: 0, hoverRadius: 6 },
    },
  };

  chart?.destroy();
  chart = new Chart(canvasEl.value, {
    type: "line",
    data,
    options,
  });
}

onMounted(build);
watch(() => props.points, build, { deep: true });
onBeforeUnmount(() => chart?.destroy());
</script>

<template>
  <div class="line-chart">
    <canvas ref="canvasEl"></canvas>

    <!-- Tooltip personnalisé -->
    <div
      v-if="tooltipVisible"
      ref="tooltipEl"
      class="custom-tooltip"
      :style="{
        left: tooltipData.x + 'px',
        top: tooltipData.y + 'px',
      }"
    >
      <div class="tooltip-date">{{ tooltipData.date }}</div>
      <div class="tooltip-price">Prix: {{ tooltipData.price }}</div>
      <div
        class="tooltip-change"
        :class="{
          positive: tooltipData.isPositive,
          negative: !tooltipData.isPositive,
        }"
      >
        {{ tooltipData.change }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.line-chart {
  width: 100%;
  height: 100%;
  background: #040d12;
  border-radius: 12px;
  overflow: hidden;
  position: relative;
}

canvas {
  display: block;
  width: 100% !important;
  height: 100% !important;
  background: transparent;
}

/* Tooltip personnalisé */
.custom-tooltip {
  position: fixed;
  background: rgba(4, 13, 18, 0.95);
  border: 1px solid #10b981;
  border-radius: 8px;
  padding: 12px;
  color: #e5e7eb;
  font-size: 12px;
  z-index: 1000;
  pointer-events: none;
  backdrop-filter: blur(8px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.tooltip-date {
  color: #e5e7eb;
  font-weight: bold;
  font-size: 13px;
  margin-bottom: 4px;
}

.tooltip-price {
  color: #9ca3af;
  margin-bottom: 2px;
}

.tooltip-change {
  font-weight: 500;
}

.tooltip-change.positive {
  color: #10b981;
}

.tooltip-change.negative {
  color: #ef4444;
}

@media (max-width: 768px) {
  .line-chart {
    border-radius: 8px;
  }
}
</style>
