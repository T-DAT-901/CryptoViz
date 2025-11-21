<script setup lang="ts">
import { onMounted, ref, watch, onBeforeUnmount, computed } from "vue";
import { useMarketStore } from "@/stores/market";
import { Chart, Tooltip, Legend, TimeScale, LinearScale } from "chart.js";
import zoomPlugin from "chartjs-plugin-zoom";
import {
  CandlestickController,
  CandlestickElement,
} from "chartjs-chart-financial";
import "chartjs-adapter-date-fns";
import type { ChartData, ChartOptions } from "chart.js";
import { fetchCandles } from "@/services/markets.api";
import type { CandleDTO } from "@/types/market";

Chart.register(
  CandlestickController,
  CandlestickElement,
  TimeScale,
  LinearScale,
  Tooltip,
  Legend,
  zoomPlugin
);

const props = defineProps<{
  candles: CandleDTO[];
  timeframe?: string;
  buildingCandle?: {
    open: number;
    high: number;
    low: number;
    close: number;
    volume: number;
  } | null;
}>();

const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart<"candlestick"> | null = null;

const tooltipVisible = ref(false);
const tooltipData = ref({
  date: "",
  open: "",
  high: "",
  low: "",
  close: "",
  change: "",
  changePercent: "",
  volume: "",
  isPositive: true,
  candleIndex: 0,
});

const crosshairVisible = ref(false);
const crosshairPosition = ref({
  x: 0,
  y: 0,
});
const crosshairLabels = ref({
  price: "",
  date: "",
  candleX: 0,
});

const tooltipPositionStyle = computed(() => {
  if (!chart || !canvasEl.value) return {};
  const element = chart.getDatasetMeta(0).data[
    tooltipData.value.candleIndex
  ] as any;
  if (!element) return {};

  const rect = canvasEl.value.getBoundingClientRect();
  const candleX = element.x;
  const candleY = element.y || 0;

  // Estimé de la taille de la tooltip (en pixels)
  const tooltipWidth = 220;
  const tooltipHeight = 150;

  let left = candleX + 15;
  let top = Math.max(10, candleY - 80);

  // Vérifier si la tooltip sort par la droite
  if (left + tooltipWidth > rect.width) {
    left = Math.max(5, candleX - tooltipWidth - 15);
  }

  // Vérifier si la tooltip sort par le bas
  if (top + tooltipHeight > rect.height) {
    top = Math.max(5, candleY - tooltipHeight - 10);
  }

  return {
    left: left + "px",
    top: top + "px",
  };
});

const options: ChartOptions<"candlestick"> = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: "nearest" },
  onHover: (event, activeElements) => {},
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false },
    zoom: {
      pan: {
        enabled: true,
        mode: "x",
        modifierKey: undefined,
        onPanRejected: ({ chart }) => {
          // Gentle limit: don't allow panning beyond data range
          const xScale = chart.scales.x as any;
          if (!xScale) return;

          // Get the min/max from the data
          const timestamps =
            props.candles.length > 0
              ? props.candles.map((c) => new Date(c.time).getTime())
              : [];

          const minTime =
            timestamps.length > 0 ? Math.min(...timestamps) : null;
          const maxTime =
            timestamps.length > 0 ? Math.max(...timestamps) : null;

          if (minTime !== null && maxTime !== null) {
            // If we've panned outside bounds, set the scale limits
            xScale.min = minTime;
            xScale.max = maxTime;
          }
        },
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
  scales: {
    x: {
      type: "time",
      grid: {
        color: "rgba(255,255,255,0.08)",
      },
      ticks: {
        color: "rgba(255,255,255,0.7)",
        font: { size: 11 },
        maxTicksLimit: 8,
      },
      time: {
        displayFormats: {
          minute: "HH:mm",
          hour: "HH:mm",
          day: "dd/MM",
          week: "dd/MM",
          month: "MMM yyyy",
          quarter: "MMM yyyy",
          year: "yyyy",
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
        font: { size: 11 },
        callback: function (value) {
          return Number(value).toLocaleString("fr-FR") + " €";
        },
      },
    },
  },
};

function getTimeDisplayFormat(timeframe: string) {
  switch (timeframe) {
    case "1h":
      return {
        unit: "minute",
        displayFormats: {
          minute: "HH:mm",
          hour: "HH:mm",
        },
        maxTicksLimit: 12,
      };
    case "1d":
      return {
        unit: "hour",
        displayFormats: {
          hour: "HH:mm",
          minute: "HH:mm",
        },
        maxTicksLimit: 12,
      };
    case "7d":
      return {
        unit: "day",
        displayFormats: {
          day: "dd/MM",
          hour: "dd/MM HH:mm",
        },
        maxTicksLimit: 7,
      };
    case "1M":
      return {
        unit: "week",
        displayFormats: {
          week: "dd/MM",
          day: "dd/MM",
        },
        maxTicksLimit: 8,
      };
    case "1y":
      return {
        unit: "month",
        displayFormats: {
          month: "MMM yyyy",
          week: "dd/MM",
        },
        maxTicksLimit: 12,
      };
    case "all":
      return {
        unit: "year",
        displayFormats: {
          year: "yyyy",
          month: "MMM yyyy",
        },
        maxTicksLimit: 10,
      };
    default:
      return {
        unit: "minute",
        displayFormats: {
          minute: "HH:mm",
          hour: "HH:mm",
        },
        maxTicksLimit: 12,
      };
  }
}

function getCandleLimit(timeframe?: string): number {
  switch (timeframe) {
    case "1m":
      return 60;
    case "5m":
      return 100;
    case "15m":
      return 80;
    case "1h":
      return 72;
    case "1d":
      return 50;
    case "7d":
    case "1w":
      return 52;
    case "1M":
      return 26;
    case "1y":
      return 52;
    default:
      return 120;
  }
}

function fitChartToTimeframe() {
  if (!chart || !props.candles?.length) return;

  chart.resetZoom();

  const timestamps = props.candles.map((c) => new Date(c.time).getTime());
  const minTime = Math.min(...timestamps);
  const maxTime = Math.max(...timestamps);

  const timeRange = maxTime - minTime;
  const margin = timeRange * 0.02;

  chart.zoomScale(
    "x",
    {
      min: minTime - margin,
      max: maxTime + margin,
    },
    "default"
  );

  const timeConfig = getTimeDisplayFormat(props.timeframe || "7d");
  if (chart.options.scales?.x) {
    const xScale = chart.options.scales.x as any;
    xScale.time = {
      ...xScale.time,
      unit: timeConfig.unit,
      displayFormats: timeConfig.displayFormats,
    };
    xScale.ticks.maxTicksLimit = timeConfig.maxTicksLimit;
    chart.update("none");
  }
}

function buildChart() {
  if (!canvasEl.value || !props.candles?.length) return;

  if (chart) {
    chart.destroy();
    chart = null;
  }

  const datasets: any[] = [
    {
      label: "BTC/USDT",
      data: props.candles.map((candle) => ({
        x: new Date(candle.time).getTime(),
        o: candle.open,
        h: candle.high,
        l: candle.low,
        c: candle.close,
      })),
      upColor: "#10b981",
      downColor: "#ef4444",
      borderUpColor: "#10b981",
      borderDownColor: "#ef4444",
      wickUpColor: "#10b981",
      wickDownColor: "#ef4444",
    } as any,
  ];

  // Ajouter la candle en construction si elle existe
  if (props.buildingCandle) {
    datasets.push({
      label: "Candle en construction",
      data: [
        {
          x: Date.now(),
          o: props.buildingCandle.open,
          h: props.buildingCandle.high,
          l: props.buildingCandle.low,
          c: props.buildingCandle.close,
        },
      ],
      upColor: "rgba(16,185,129,0.4)",
      downColor: "rgba(239,68,68,0.4)",
      borderUpColor: "rgba(16,185,129,0.6)",
      borderDownColor: "rgba(239,68,68,0.6)",
      wickUpColor: "rgba(16,185,129,0.6)",
      wickDownColor: "rgba(239,68,68,0.6)",
    } as any);
  }

  const chartData: ChartData<"candlestick"> = {
    datasets,
  };

  chart = new Chart(canvasEl.value, {
    type: "candlestick",
    data: chartData,
    options: options,
  });

  setTimeout(() => {
    fitChartToTimeframe();
  }, 100);

  canvasEl.value.addEventListener("mousemove", handleMouseMove);
  canvasEl.value.addEventListener("mouseleave", () => {
    tooltipVisible.value = false;
    crosshairVisible.value = false;
  });
}

function handleMouseMove(event: MouseEvent) {
  if (!chart || !canvasEl.value) return;

  const rect = canvasEl.value.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;

  crosshairVisible.value = true;

  const elements = chart.getElementsAtEventForMode(
    event,
    "nearest",
    { intersect: false },
    false
  );

  if (elements.length > 0) {
    const dataIndex = elements[0].index;
    const candle = props.candles[dataIndex];
    const element = elements[0].element as any;

    if (candle && element) {
      const candleCenterX = element.x;

      const yScale = chart.scales.y;
      const priceAtMouseY = yScale.getValueForPixel(y);

      crosshairPosition.value = { x: candleCenterX, y };

      crosshairLabels.value = {
        price:
          (priceAtMouseY || 0).toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        date: new Date(candle.time).toLocaleDateString("fr-FR", {
          day: "2-digit",
          month: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
        }),
        candleX: candleCenterX,
      };

      const change = candle.close - candle.open;
      const changePercent = (change / candle.open) * 100;

      tooltipData.value = {
        date: new Date(candle.time).toLocaleDateString("fr-FR", {
          day: "2-digit",
          month: "2-digit",
          year: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        }),
        open:
          candle.open.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        high:
          candle.high.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        low:
          candle.low.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        close:
          candle.close.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        change: `${change > 0 ? "+" : ""}${change.toFixed(2)} €`,
        changePercent: `${change > 0 ? "+" : ""}${changePercent.toFixed(2)}%`,
        volume: "Volume: N/A",
        isPositive: change >= 0,
        candleIndex: dataIndex,
      };

      tooltipVisible.value = true;
    }
  } else {
    crosshairPosition.value = { x, y };
    tooltipVisible.value = false;

    const yScale = chart.scales.y;
    const priceAtMouseY = yScale.getValueForPixel(y);

    crosshairLabels.value.price =
      (priceAtMouseY || 0).toLocaleString("fr-FR", {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }) + " €";
  }
}

onMounted(() => {
  console.log("✅ CandleChart mounted");
  buildChart();
});

watch(() => props.candles, buildChart, { deep: true });

watch(
  () => props.buildingCandle,
  (newBuilding) => {
    console.log("🔄 CandleChart: buildingCandle changed", newBuilding);
    if (newBuilding) {
      console.log("  → Rebuilding chart with building candle");
      buildChart();
    }
  },
  { deep: true }
);
watch(
  () => props.timeframe,
  () => {
    setTimeout(() => {
      fitChartToTimeframe();
    }, 100);
  },
  { immediate: false }
);
onBeforeUnmount(() => {
  if (canvasEl.value) {
    canvasEl.value.removeEventListener("mousemove", handleMouseMove);
  }
  chart?.destroy();
});

function resetZoom() {
  if (chart) {
    chart.resetZoom();
  }
}

defineExpose({
  chart,
  resetZoom,
  fitChartToTimeframe,
});
</script>

<template>
  <div class="candle-chart">
    <button
      class="candle-chart-reset-btn"
      @click="resetZoom"
      title="Reset zoom (Ctrl+scroll to zoom)"
    >
      🔍 Reset
    </button>

    <div class="candle-chart-canvas-wrapper">
      <canvas ref="canvasEl"></canvas>
    </div>

    <div v-if="crosshairVisible" class="candle-chart-crosshair-container">
      <div
        class="candle-chart-crosshair-line candle-chart-crosshair-line--vertical"
        :style="{ left: crosshairPosition.x + 'px' }"
      >
        <div class="candle-chart-crosshair-label candle-chart-date-label">
          {{ crosshairLabels.date }}
        </div>
      </div>

      <div
        class="candle-chart-crosshair-line candle-chart-crosshair-line--horizontal"
        :style="{ top: crosshairPosition.y + 'px' }"
      >
        <div class="candle-chart-crosshair-label candle-chart-price-label">
          {{ crosshairLabels.price }}
        </div>
      </div>
    </div>

    <div
      v-if="tooltipVisible"
      class="candle-chart-tooltip"
      :style="tooltipPositionStyle"
    >
      <div class="candle-chart-tooltip-date">{{ tooltipData.date }}</div>
      <div class="candle-chart-tooltip-ohlc">
        <div class="candle-chart-ohlc-columns">
          <!-- Colonne gauche: Open et Close -->
          <div class="candle-chart-ohlc-column">
            <div class="candle-chart-ohlc-row">
              <span class="candle-chart-ohlc-label">Open:</span>
              <span class="candle-chart-ohlc-value">{{
                tooltipData.open
              }}</span>
            </div>
            <div class="candle-chart-ohlc-row">
              <span class="candle-chart-ohlc-label">Close:</span>
              <span class="candle-chart-ohlc-value">{{
                tooltipData.close
              }}</span>
            </div>
          </div>
          <!-- Colonne droite: Low et High -->
          <div class="candle-chart-ohlc-column">
            <div class="candle-chart-ohlc-row">
              <span class="candle-chart-ohlc-label">Low:</span>
              <span
                class="candle-chart-ohlc-value candle-chart-ohlc-value--low"
                >{{ tooltipData.low }}</span
              >
            </div>
            <div class="candle-chart-ohlc-row">
              <span class="candle-chart-ohlc-label">High:</span>
              <span
                class="candle-chart-ohlc-value candle-chart-ohlc-value--high"
                >{{ tooltipData.high }}</span
              >
            </div>
          </div>
        </div>
      </div>
      <div class="candle-chart-tooltip-change">
        <span
          :class="{
            'candle-chart-change--positive': tooltipData.isPositive,
            'candle-chart-change--negative': !tooltipData.isPositive,
          }"
        >
          {{ tooltipData.change }} ({{ tooltipData.changePercent }})
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.candle-chart {
  position: relative;
  width: 100%;
  height: 100%;
}

.candle-chart-canvas-wrapper {
  width: 100%;
  height: 100%;
}

.candle-chart-reset-btn {
  position: absolute;
  top: 10px;
  right: 10px;
  z-index: 10;
  padding: 6px 12px;
  background: rgba(16, 185, 129, 0.2);
  border: 1px solid rgba(16, 185, 129, 0.5);
  color: #10b981;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.2s ease;
}

.candle-chart-reset-btn:hover {
  background: rgba(16, 185, 129, 0.4);
  border-color: #10b981;
}

.candle-chart-crosshair-container {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.candle-chart-crosshair-line {
  position: absolute;
  opacity: 0.5;
  pointer-events: none;
}

.candle-chart-crosshair-line--vertical {
  top: 0;
  width: 1px;
  height: 100%;
  border-left: 1px dashed rgba(255, 255, 255, 0.3);
}

.candle-chart-crosshair-line--horizontal {
  left: 0;
  width: 100%;
  height: 1px;
  border-top: 1px dashed rgba(255, 255, 255, 0.3);
}

.candle-chart-crosshair-label {
  position: absolute;
  background: rgba(31, 41, 55, 0.9);
  color: rgba(255, 255, 255, 0.8);
  padding: 4px 8px;
  font-size: 11px;
  border-radius: 3px;
  white-space: nowrap;
  z-index: 20;
}

.candle-chart-date-label {
  left: 50%;
  transform: translateX(-50%);
  bottom: -24px;
}

.candle-chart-price-label {
  right: -8px;
  top: 50%;
  transform: translateY(-50%);
  margin-right: -60px;
}

.candle-chart-tooltip {
  position: absolute;
  background: rgba(7, 14, 16, 0.95);
  border: 1px solid rgba(232, 240, 240, 0.15);
  border-radius: 4px;
  padding: 7px 9px;
  z-index: 1000;
  color: rgba(232, 240, 240, 0.9);
  font-size: 9px;
  pointer-events: none;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  max-width: 200px;
  backdrop-filter: blur(8px);
  white-space: nowrap;
}

.candle-chart-tooltip-date {
  font-weight: 600;
  margin-bottom: 6px;
  color: rgba(232, 240, 240, 0.95);
  border-bottom: 1px solid rgba(232, 240, 240, 0.1);
  padding-bottom: 4px;
  font-size: 9px;
  text-align: center;
}

.candle-chart-tooltip-ohlc {
  margin-bottom: 6px;
}

.candle-chart-ohlc-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.candle-chart-ohlc-column {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.candle-chart-ohlc-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 9px;
  gap: 4px;
}

.candle-chart-ohlc-label {
  color: rgba(232, 240, 240, 0.6);
  font-weight: 500;
  min-width: 40px;
  text-align: left;
  flex-shrink: 0;
}

.candle-chart-ohlc-value {
  color: rgba(232, 240, 240, 0.9);
  font-family: "Monaco", "Courier New", monospace;
  text-align: right;
  font-size: 9px;
  font-weight: 500;
  flex-shrink: 0;
}

.candle-chart-ohlc-value--high {
  color: #10b981;
}

.candle-chart-ohlc-value--low {
  color: #ef4444;
}

.candle-chart-tooltip-change {
  text-align: center;
  font-weight: 600;
  font-size: 9px;
  padding-top: 4px;
  border-top: 1px solid rgba(232, 240, 240, 0.1);
}

.candle-chart-change--positive {
  color: #10b981;
}

.candle-chart-change--negative {
  color: #ef4444;
}
</style>
