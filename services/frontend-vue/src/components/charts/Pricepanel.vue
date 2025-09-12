<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import CandleChart from "@/components/charts/CandleChart.vue";
import LineChart from "@/components/charts/LineChart.vue";
import { useMarketStore } from "@/stores/market";
import { fetchCandles } from "@/services/markets.api";
import type { CandleDTO } from "@/types/market";

const store = useMarketStore();
const mode = ref<"candle" | "line">("candle");
const loading = ref(false);

// source data for both views
const candles = ref<CandleDTO[]>([]);

const linePoints = computed(() =>
  candles.value.map((c) => ({
    // support t (ms) ou ts (ms) — et fallback Date si jamais
    x:
      typeof (c as any).ts === "number"
        ? (c as any).ts
        : typeof (c as any).t === "number"
          ? (c as any).t
          : new Date((c as any).t).getTime(),
    y: c.c,
  }))
);

async function load() {
  loading.value = true;
  try {
    const rows = await fetchCandles("BTC", store.interval, 300);
    candles.value = rows;
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <section class="panel">
    <div class="toolbar">
      <div class="btns">
        <button
          :class="['tab', { active: mode === 'line' }]"
          @click="mode = 'line'"
        >
          Prix
        </button>
        <button
          :class="['tab', { active: mode === 'candle' }]"
          @click="mode = 'candle'"
        >
          Chandeliers
        </button>
      </div>

      <div class="right">
        <span class="tf">{{ store.interval }}</span>
      </div>
    </div>

    <div class="chart-wrap">
      <div v-if="loading" class="loading">Chargement…</div>

      <!-- show one or the other; unmounts the other chart cleanly -->
      <LineChart v-if="!loading && mode === 'line'" :points="linePoints" />
      <CandleChart
        v-else-if="!loading && mode === 'candle'"
        :candles="candles"
      />
    </div>
  </section>
</template>

<style scoped>
.panel {
  background: #222;
  border: 1px solid #2f2f2f;
  border-radius: 12px;
  padding: 12px;
}
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.btns {
  display: flex;
  gap: 8px;
}
.tab {
  padding: 6px 12px;
  border-radius: 8px;
  background: #2a2a2a;
  border: 1px solid #3a3a3a;
  color: #cfd3d7;
  cursor: pointer;
}
.tab.active {
  color: #e9ecef;
  border-color: #4b87ff;
  background: #2b2f3a;
}
.chart-wrap {
  height: 460px;
}
.loading {
  color: #9aa3ab;
  padding: 16px;
}
.tf {
  color: #aab2b9;
  font-size: 12px;
}
</style>
