<script setup lang="ts">
import { onMounted, ref, watch, onBeforeUnmount } from "vue";
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

// Register Chart.js controllers/elements once
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
  timeframe?: string; // Ajouter le timeframe pour ajuster les échelles
}>();

const canvasEl = ref<HTMLCanvasElement | null>(null);
let chart: Chart<"candlestick"> | null = null;

// État du tooltip personnalisé
const tooltipVisible = ref(false);
const tooltipData = ref({
  x: 0,
  y: 0,
  date: "",
  open: "",
  high: "",
  low: "",
  close: "",
  change: "",
  changePercent: "",
  volume: "",
  isPositive: true,
});

// État des lignes de repère (crosshair)
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

const options: ChartOptions<"candlestick"> = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: "nearest" },
  onHover: (event, activeElements) => {
    // Supprimé le log de debug
  },
  plugins: {
    legend: { display: false },
    tooltip: { enabled: false }, // Désactiver le tooltip par défaut
    zoom: {
      pan: {
        enabled: true,
        mode: "x", // Seulement horizontal
        modifierKey: "shift", // Maintenir Shift pour pan
      },
      zoom: {
        wheel: {
          enabled: true,
          modifierKey: "ctrl", // Ctrl + scroll pour zoom
          speed: 0.05, // Vitesse de zoom réduite (par défaut: 0.1)
        },
        pinch: {
          enabled: true,
        },
        mode: "x", // Zoom horizontal seulement
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
        maxTicksLimit: 8, // Limite le nombre de ticks pour éviter l'encombrement
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

// Fonction pour obtenir le format temporel selon le timeframe
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

// Fonction pour ajuster automatiquement le zoom sur la période
function fitChartToTimeframe() {
  if (!chart || !props.candles?.length) return;

  // Réinitialiser le zoom d'abord
  chart.resetZoom();

  // Obtenir les timestamps min et max des données
  const timestamps = props.candles.map((c) => c.t);
  const minTime = Math.min(...timestamps);
  const maxTime = Math.max(...timestamps);

  // Ajouter une petite marge (2% de chaque côté)
  const timeRange = maxTime - minTime;
  const margin = timeRange * 0.02;

  // Appliquer le zoom pour s'adapter exactement aux données
  chart.zoomScale(
    "x",
    {
      min: minTime - margin,
      max: maxTime + margin,
    },
    "default"
  );

  // Mettre à jour les formats d'affichage selon le timeframe
  const timeConfig = getTimeDisplayFormat(props.timeframe || "7d");
  if (chart.options.scales?.x) {
    const xScale = chart.options.scales.x as any;
    xScale.time = {
      ...xScale.time,
      unit: timeConfig.unit,
      displayFormats: timeConfig.displayFormats,
    };
    xScale.ticks.maxTicksLimit = timeConfig.maxTicksLimit;
    chart.update("none"); // Mise à jour sans animation pour la fluidité
  }
}

// Données et construction du graphique
function buildChart() {
  if (!canvasEl.value || !props.candles?.length) return;

  if (chart) {
    chart.destroy();
    chart = null;
  }

  const chartData: ChartData<"candlestick"> = {
    datasets: [
      {
        label: "BTC/USDT",
        data: props.candles.map((candle) => ({
          x: candle.t,
          o: candle.o,
          h: candle.h,
          l: candle.l,
          c: candle.c,
        })),
        upColor: "#10b981",
        downColor: "#ef4444",
        borderUpColor: "#10b981",
        borderDownColor: "#ef4444",
        wickUpColor: "#10b981",
        wickDownColor: "#ef4444",
      } as any,
    ],
  };

  chart = new Chart(canvasEl.value, {
    type: "candlestick",
    data: chartData,
    options: options,
  });

  // Ajuster automatiquement le zoom et les échelles
  setTimeout(() => {
    fitChartToTimeframe();
  }, 100); // Petit délai pour que le chart soit complètement initialisé

  // Ajouter un événement mousemove sur le canvas
  canvasEl.value.addEventListener("mousemove", handleMouseMove);
  canvasEl.value.addEventListener("mouseleave", () => {
    tooltipVisible.value = false;
    crosshairVisible.value = false;
  });
}

// Gestionnaire d'événement mousemove personnalisé
function handleMouseMove(event: MouseEvent) {
  if (!chart || !canvasEl.value) return;

  const rect = canvasEl.value.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;

  crosshairVisible.value = true;

  // Obtenir les éléments à cette position (la bougie la plus proche)
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
      // Obtenir la position X exacte du centre de la bougie (mèche)
      const candleCenterX = element.x;

      // Calculer le prix à la position Y de la souris
      const yScale = chart.scales.y;
      const priceAtMouseY = yScale.getValueForPixel(y);

      // Mettre à jour les positions du crosshair
      crosshairPosition.value = { x: candleCenterX, y };

      // Mettre à jour les labels
      crosshairLabels.value = {
        price:
          (priceAtMouseY || 0).toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        date: new Date(candle.t).toLocaleDateString("fr-FR", {
          day: "2-digit",
          month: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
        }),
        candleX: candleCenterX,
      };

      // Calculer la variation pour le tooltip
      const change = candle.c - candle.o;
      const changePercent = (change / candle.o) * 100;

      tooltipData.value = {
        x: event.clientX,
        y: event.clientY - 120,
        date: new Date(candle.t).toLocaleDateString("fr-FR", {
          day: "2-digit",
          month: "2-digit",
          year: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        }),
        open:
          candle.o.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        high:
          candle.h.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        low:
          candle.l.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        close:
          candle.c.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          }) + " €",
        change: `${change > 0 ? "+" : ""}${change.toFixed(2)} €`,
        changePercent: `${change > 0 ? "+" : ""}${changePercent.toFixed(2)}%`,
        volume: "Volume: N/A",
        isPositive: change >= 0,
      };

      tooltipVisible.value = true;
    }
  } else {
    // Si pas de bougie trouvée, on suit quand même la souris pour la ligne horizontale
    crosshairPosition.value = { x, y };
    tooltipVisible.value = false;

    // Calculer le prix pour la ligne horizontale
    const yScale = chart.scales.y;
    const priceAtMouseY = yScale.getValueForPixel(y);

    crosshairLabels.value.price =
      (priceAtMouseY || 0).toLocaleString("fr-FR", {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }) + " €";
  }
}

// Lifecycle
onMounted(buildChart);
watch(() => props.candles, buildChart, { deep: true });
watch(
  () => props.timeframe,
  () => {
    // Quand le timeframe change, ajuster le zoom
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

// Fonction pour réinitialiser le zoom
function resetZoom() {
  if (chart) {
    chart.resetZoom();
  }
}

// Exposer les méthodes pour le composant parent
defineExpose({
  chart,
  resetZoom,
  fitChartToTimeframe,
});
</script>

<template>
  <div class="candle-chart">
    <!-- Bouton de reset du zoom -->
    <button
      class="reset-zoom-btn"
      @click="resetZoom"
      title="Réinitialiser le zoom (Ctrl+scroll pour zoomer)"
    >
      🔍 Reset
    </button>

    <canvas ref="canvasEl"></canvas>

    <!-- Lignes de repère (crosshair) -->
    <div v-if="crosshairVisible" class="crosshair-container">
      <!-- Ligne verticale (fixée sur la mèche de la bougie) -->
      <div
        class="crosshair-line vertical"
        :style="{ left: crosshairPosition.x + 'px' }"
      >
        <!-- Label de date en bas -->
        <div class="crosshair-label date-label">
          {{ crosshairLabels.date }}
        </div>
      </div>

      <!-- Ligne horizontale -->
      <div
        class="crosshair-line horizontal"
        :style="{ top: crosshairPosition.y + 'px' }"
      >
        <!-- Label de prix à droite -->
        <div class="crosshair-label price-label">
          {{ crosshairLabels.price }}
        </div>
      </div>
    </div>

    <!-- Tooltip personnalisé -->
    <div
      v-if="tooltipVisible"
      class="custom-tooltip"
      :style="{
        left: tooltipData.x + 'px',
        top: tooltipData.y + 'px',
      }"
    >
      <div class="tooltip-date">{{ tooltipData.date }}</div>
      <div class="tooltip-ohlc">
        <div class="ohlc-row">
          <span class="label">O:</span>
          <span class="value">{{ tooltipData.open }}</span>
        </div>
        <div class="ohlc-row">
          <span class="label">H:</span>
          <span class="value high">{{ tooltipData.high }}</span>
        </div>
        <div class="ohlc-row">
          <span class="label">L:</span>
          <span class="value low">{{ tooltipData.low }}</span>
        </div>
        <div class="ohlc-row">
          <span class="label">C:</span>
          <span class="value">{{ tooltipData.close }}</span>
        </div>
      </div>
      <div class="tooltip-change">
        <span
          :class="{
            positive: tooltipData.isPositive,
            negative: !tooltipData.isPositive,
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

/* Bouton de reset du zoom */
.reset-zoom-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 10;
  background: rgba(4, 13, 18, 0.9);
  border: 1px solid #374151;
  border-radius: 6px;
  color: #9ca3af;
  padding: 6px 10px;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.2s ease;
  backdrop-filter: blur(8px);
}

.reset-zoom-btn:hover {
  background: #374151;
  color: #e5e7eb;
}

/* Lignes de repère (crosshair) */
.crosshair-container {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
  z-index: 5;
}

.crosshair-line {
  position: absolute;
  pointer-events: none;
  /* opacity: 0.7; */
  transition: opacity 0.15s ease;
}

.crosshair-line.vertical {
  width: 0;
  height: 100%;
  border-left: 1px dashed rgba(229, 231, 235, 0.4);
  filter: drop-shadow(0 0 1px rgba(229, 231, 235, 0.2));
}

.crosshair-line.horizontal {
  width: 100%;
  height: 0;
  border-top: 1px dashed rgba(229, 231, 235, 0.4);
  filter: drop-shadow(0 0 1px rgba(229, 231, 235, 0.2));
}

/* Animation douce à l'apparition */
.crosshair-container {
  animation: fadeIn 0.2s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

/* Labels des lignes de repère */
.crosshair-label {
  position: absolute;
  background: rgba(4, 13, 18, 0.95);
  border: 1px solid #374151;
  border-radius: 4px;
  padding: 4px 8px;
  color: #e5e7eb;
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
  backdrop-filter: blur(8px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.date-label {
  bottom: 0px;
  left: 50%;
  transform: translateX(-50%);
  background: rgb(16, 185, 129);
  border-color: #10b981;
  color: #040d12;
}

.price-label {
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: rgb(229, 231, 235) !important;
  border-color: #9ca3af;
  color: #040d12;
}

/* Tooltip personnalisé */
.custom-tooltip {
  position: fixed;
  background: rgba(4, 13, 18, 0.95);
  border: 1px solid #374151;
  border-radius: 8px;
  padding: 12px;
  color: #e5e7eb;
  font-size: 12px;
  z-index: 1000;
  pointer-events: none;
  /* backdrop-filter: blur(8px); */
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  min-width: 180px;
}

.tooltip-date {
  color: #e5e7eb;
  font-weight: bold;
  font-size: 13px;
  margin-bottom: 8px;
  text-align: center;
}

.tooltip-ohlc {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  margin-bottom: 8px;
}

.ohlc-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ohlc-row .label {
  color: #9ca3af;
  font-weight: 600;
  width: 20px;
}

.ohlc-row .value {
  color: #e5e7eb;
  font-weight: 500;
}

.ohlc-row .value.high {
  color: #10b981;
}

.ohlc-row .value.low {
  color: #ef4444;
}

.tooltip-change {
  text-align: center;
  font-weight: 600;
  padding-top: 4px;
  border-top: 1px solid #374151;
}

.tooltip-change .positive {
  color: #10b981;
}

.tooltip-change .negative {
  color: #ef4444;
}

@media (max-width: 768px) {
  .candle-chart {
    border-radius: 8px;
  }

  .reset-zoom-btn {
    top: 8px;
    right: 8px;
    padding: 4px 8px;
    font-size: 10px;
  }

  .custom-tooltip {
    min-width: 160px;
    font-size: 11px;
  }
}
</style>
