<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from "vue";
import {
  Chart,
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
let chart: Chart<"line"> | null = null;

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
  grad.addColorStop(0, "rgba(16,185,129,0.35)");
  grad.addColorStop(1, "rgba(16,185,129,0)");

  const data: ChartData<"line"> = {
    datasets: [
      {
        label: "Prix",
        // IMPORTANT: give numeric x to satisfy (number|Point|null)[]
        data: toNumericPoints(props.points),
        parsing: false, // we already provide {x,y}
        borderColor: "#22c55e",
        backgroundColor: grad,
        borderWidth: 2,
        fill: true,
        tension: 0.25,
        pointRadius: 0,
      },
    ],
  };

  const options: ChartOptions<"line"> = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: { intersect: false, mode: "index" },
    scales: {
      x: {
        type: "time",
        grid: { color: "rgba(255,255,255,0.05)" },
        ticks: { color: "rgba(220,220,220,0.7)" },
      },
      y: {
        position: "right",
        grid: { color: "rgba(255,255,255,0.05)" },
        ticks: { color: "rgba(220,220,220,0.7)" },
      },
    },
    plugins: {
      legend: { display: false },
      tooltip: { mode: "index", intersect: false },
    },
    elements: {
      point: { radius: 0, hoverRadius: 6 },
    },
  };

  chart?.destroy();
  chart = new Chart(canvasEl.value, { type: "line", data, options });
}

onMounted(build);
watch(() => props.points, build, { deep: true });
onBeforeUnmount(() => chart?.destroy());
</script>

<template>
  <canvas ref="canvasEl"></canvas>
</template>

<style scoped>
:host,
canvas {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
