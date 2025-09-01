<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useMarketStore } from "@/stores/market";
import { fetchRSI } from "@/services/indicators.api";
import type { RSIPoint } from "@/types/market";

import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Legend,
} from "chart.js";
import type { ChartData, ChartOptions } from "chart.js";
import "chartjs-adapter-date-fns";

// register needed pieces once
Chart.register(
  LineElement,
  LineController,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Legend
);

const market = useMarketStore();

// canvas + chart instance
const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart<"line"> | null = null;

// data & options
const dataRef = ref<ChartData<"line">>({
  labels: [],
  datasets: [
    {
      label: "RSI",
      data: [], // numbers 0..100
      borderColor: "#22c55e",
      backgroundColor: "rgba(34,197,94,0.25)",
      borderWidth: 1.5,
      pointRadius: 0,
      tension: 0.2,
    },
    // 70 band
    {
      label: "70",
      data: [],
      borderColor: "rgba(239,68,68,0.8)",
      borderDash: [6, 6],
      pointRadius: 0,
      borderWidth: 1,
    },
    // 30 band
    {
      label: "30",
      data: [],
      borderColor: "rgba(34,197,94,0.8)",
      borderDash: [6, 6],
      pointRadius: 0,
      borderWidth: 1,
    },
  ],
});

const options = ref<ChartOptions<"line">>({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    mode: "nearest",
    intersect: false,
    axis: "x",
  },
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false },
  },
  scales: {
    x: {
      type: "time",
      grid: { color: "rgba(255,255,255,0.06)" },
      ticks: { color: "rgba(255,255,255,0.55)" },
    },
    y: {
      min: 0,
      max: 100,
      position: "right",
      grid: { color: "rgba(255,255,255,0.06)" },
      ticks: { color: "rgba(255,255,255,0.65)" },
    },
  },
});

async function load() {
  // read from mock via indicators.api
  const rows: RSIPoint[] = await fetchRSI("BTC", market.interval, 14);
  const xs = rows.map((r) => r.ts);
  const ys = rows.map((r) => r.value);

  dataRef.value.labels = xs;
  dataRef.value.datasets[0].data = ys;
  dataRef.value.datasets[1].data = xs.map(() => 70);
  dataRef.value.datasets[2].data = xs.map(() => 30);

  if (chart) chart.update();
}

function build() {
  if (!canvasEl.value) return;
  if (chart) chart.destroy();
  chart = new Chart(canvasEl.value, {
    type: "line",
    data: dataRef.value,
    options: options.value,
  });
}

// Helper to draw rounded rectangles
function roundRectPath(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number
) {
  const rr = Math.min(r, h / 2, w / 2);
  ctx.beginPath();
  ctx.moveTo(x + rr, y);
  ctx.arcTo(x + w, y, x + w, y + h, rr);
  ctx.arcTo(x + w, y + h, x, y + h, rr);
  ctx.arcTo(x, y + h, x, y, rr);
  ctx.arcTo(x, y, x + w, y, rr);
  ctx.closePath();
}

// Custom overlay: vertical guide + point + value bubble
const rsiHoverOverlay = {
  id: "rsiHoverOverlay",
  afterDraw(chart: any) {
    const active = chart.getActiveElements?.();
    if (!active || active.length === 0) return;

    const { ctx, chartArea } = chart;
    const { datasetIndex, index } = active[0];
    const meta = chart.getDatasetMeta(datasetIndex);
    const pt = meta.data[index];
    if (!pt) return;

    // Values
    const val = chart.data.datasets[datasetIndex].data[index] as number;
    const label =
      typeof val === "number" ? `RSI ${val.toFixed(2)}` : `RSI ${val}`;

    // Neutral colors (adjust later if needed)
    const guideColor = "rgba(220, 220, 220, 0.50)"; // soft gray
    const dotColor = "#d1d5db"; // light gray
    const bubbleBg = "#111827"; // dark slate
    const bubbleText = "#e5e7eb"; // near-white
    const bubbleBorder = "#94a3b8"; // muted border

    // Vertical guide (thin line + soft column)
    ctx.save();
    ctx.fillStyle = "rgba(148,163,184,0.08)";
    ctx.fillRect(
      pt.x - 14,
      chartArea.top,
      28,
      chartArea.bottom - chartArea.top
    );
    ctx.strokeStyle = guideColor;
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(pt.x, chartArea.top);
    ctx.lineTo(pt.x, chartArea.bottom);
    ctx.stroke();
    ctx.restore();

    // Point
    ctx.save();
    ctx.fillStyle = dotColor;
    ctx.beginPath();
    ctx.arc(pt.x, pt.y, 4, 0, Math.PI * 2);
    ctx.fill();
    ctx.restore();

    // Bubble
    ctx.save();
    ctx.font = "12px ui-sans-serif, -apple-system, Segoe UI, Roboto, Ubuntu";
    const padX = 10,
      padY = 6,
      h = 28;
    const textW = ctx.measureText(label).width;
    const w = textW + padX * 2;
    const x = Math.min(
      Math.max(pt.x - w / 2, chartArea.left + 6),
      chartArea.right - w - 6
    );
    const y = pt.y - h - 10;

    roundRectPath(ctx, x, y, w, h, 10);
    ctx.fillStyle = bubbleBg;
    ctx.fill();
    ctx.lineWidth = 1;
    ctx.strokeStyle = bubbleBorder;
    ctx.stroke();

    ctx.fillStyle = bubbleText;
    ctx.textBaseline = "middle";
    ctx.fillText(label, x + padX, y + h / 2);
    ctx.restore();
  },
};
Chart.register(rsiHoverOverlay);

onMounted(async () => {
  await load();
  build();
});

// reload when timeframe changes
watch(
  () => market.interval,
  async () => {
    await load();
  }
);
</script>

<template>
  <section class="panel">
    <div class="toolbar">
      <div class="left">
        <span class="title">RSI (14)</span>
      </div>
    </div>
    <div class="chart-wrap">
      <canvas ref="canvasEl"></canvas>
    </div>
  </section>
</template>

<style scoped>
.panel {
  background: #0b0e11;
  color: #e5e7eb;
  border-radius: 10px;
  border: 1px solid #1a1f24;
  padding: 12px;
  margin-top: 12px;
}
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: #9ca3af;
  font-size: 14px;
}
.left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.title {
  font-weight: 700;
  color: #e5e7eb;
}
.chart-wrap {
  height: 220px;
} /* typical RSI panel height */
</style>
