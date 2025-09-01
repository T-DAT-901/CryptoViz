<script setup lang="ts">
import { Line } from "vue-chartjs";
import {
  Chart,
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Legend,
} from "chart.js";
import "chartjs-adapter-date-fns";
import { onMounted, ref, watch } from "vue";
import { fetchRSI } from "@/services/indicators.api";
import { useMarketStore } from "@/stores/market";
import { useIndicatorsStore } from "@/stores/indicators";
import type { ChartOptions } from "chart.js";

Chart.register(
  LineElement,
  PointElement,
  LinearScale,
  TimeScale,
  Tooltip,
  Legend
);

const props = defineProps<{ symbol: string }>();
const market = useMarketStore();
const ind = useIndicatorsStore();

const dataRef = ref({
  labels: [] as number[],
  datasets: [{ label: "RSI", data: [] as number[] }],
});
const options = ref<ChartOptions<"line">>({
  responsive: true,
  scales: {
    x: {
      type: "time",
    },
    y: {
      min: 0,
      max: 100,
    },
  },
});

async function load() {
  const rows = await fetchRSI(props.symbol, market.interval, ind.rsiPeriod);
  dataRef.value.labels = rows.map((r) => r.ts);
  dataRef.value.datasets[0].data = rows.map((r) => r.value);
}
watch(() => [market.interval, ind.rsiPeriod], load, { deep: true });
onMounted(load);
</script>

<template>
  <Line :data="dataRef" :options="options" />
</template>
