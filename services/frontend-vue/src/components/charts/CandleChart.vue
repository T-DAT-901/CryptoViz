<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useMarketStore } from "@/stores/market";
import { Chart, Tooltip, Legend, TimeScale, LinearScale } from "chart.js";
import {
  CandlestickController,
  CandlestickElement,
} from "chartjs-chart-financial";
import "chartjs-adapter-date-fns";
import type { ChartData, ChartOptions } from "chart.js";
import { fetchCandles } from "@/services/markets.api";
import type { CandleDTO } from "@/types/market";

// Register Chart.js controllers/elements once
Chart.register(
  CandlestickController,
  CandlestickElement,
  TimeScale,
  LinearScale,
  Tooltip,
  Legend
);

const store = useMarketStore();
const timeframes = ["1m", "30m", "1h"] as const;
const volumeText = ref("$223K");

// Canvas + chart instance
const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart<"candlestick"> | null = null;

// Chart data
const dataRef = ref<ChartData<"candlestick">>({
  datasets: [
    {
      label: "BTC/USDT",
      data: [] as any[], // [{ x, o, h, l, c }]
      // Candlestick colors (match TradingView-like palette)
      upColor: "#16a34a",
      downColor: "#ef4444",
      borderUpColor: "#16a34a",
      borderDownColor: "#ef4444",
      wickUpColor: "#16a34a",
      wickDownColor: "#ef4444",
    } as any, // candlestick dataset options are plugin-augmented
  ],
});

// Chart options
const options = ref<ChartOptions<"candlestick">>({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: { mode: "index", intersect: false },
  },
  scales: {
    x: {
      type: "time",
      grid: { color: "rgba(255,255,255,0.06)" },
      ticks: { color: "rgba(255,255,255,0.55)" },
    },
    y: {
      position: "right",
      grid: { color: "rgba(255,255,255,0.06)" },
      ticks: { color: "rgba(255,255,255,0.65)" },
    },
  },
});

async function load() {
  // Use mocks when VITE_USE_MOCK=true
  const rows: CandleDTO[] = await fetchCandles("BTC", store.interval, 120);
  dataRef.value.datasets[0].data = rows.map((r) => ({
    x: r.t,
    o: r.o,
    h: r.h,
    l: r.l,
    c: r.c,
  }));
  // Update chart if already built
  if (chart) chart.update();
}

function buildChart() {
  if (!canvasEl.value) return;
  if (chart) {
    chart.destroy();
    chart = null;
  }
  chart = new Chart(canvasEl.value, {
    type: "candlestick",
    data: dataRef.value,
    options: options.value,
  });
}

// Initial load + build
onMounted(async () => {
  await load();
  buildChart();
});

// Reload data when timeframe changes
watch(
  () => store.interval,
  async () => {
    await load();
  }
);
</script>

<template>
  <section class="panel">
    <div class="toolbar">
      <div class="left">
        <span class="pair">BTC/USDT</span>
        <span class="dot">•</span>
        <span class="tf nowrap">
          <button
            v-for="tf in timeframes"
            :key="tf"
            :class="['tf-btn', { active: store.interval === tf }]"
            @click="store.setInterval(tf)"
          >
            {{ tf }}
          </button>
        </span>
        <span class="dot">•</span>
        <span class="ex">Binance</span>
      </div>
    </div>

    <div class="sub">
      Volume <span class="g">{{ volumeText }}</span>
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
  flex-wrap: wrap;
}
.pair {
  font-weight: 700;
  color: #e5e7eb;
  letter-spacing: 0.3px;
}
.dot {
  opacity: 0.4;
}
.nowrap {
  white-space: nowrap;
}

.tf-btn {
  background: #12161c;
  color: #9ca3af;
  border: 1px solid #1f2937;
  padding: 6px 10px;
  border-radius: 6px;
  margin-right: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
}
.tf-btn:hover {
  color: #e5e7eb;
  border-color: #374151;
}
.tf-btn.active {
  color: #e5e7eb;
  border-color: #2563eb;
  background: #111827;
}

.ex {
  color: #d1d5db;
}
.sub {
  margin: 8px 0 6px;
  font-size: 13px;
  color: #9ca3af;
}
.g {
  color: #16a34a;
  font-weight: 600;
}

.chart-wrap {
  height: 440px;
}
</style>
