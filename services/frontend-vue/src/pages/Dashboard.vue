<script setup lang="ts">
import CandleChart from "@/components/charts/CandleChart.vue";
import IndicatorsPanel from "@/components/IndicatorsPanel.vue";
import { onMounted } from "vue";
import { useMarketStore } from "@/stores/market";
import { fetchCandles } from "@/services/markets.api";
import RSIChart from "@/components/charts/RSIChart.vue";

const store = useMarketStore();

// Preload initial candles so the chart has data immediately
onMounted(async () => {
  const rows = await fetchCandles("BTC", store.interval, 120);
  store.setCandles?.("BTC", rows); // if your store has per-symbol
  // If you switched to BTC-only store, use: store.candles = rows
});
</script>

<template>
  <main style="padding: 16px; max-width: 1280px; margin: auto">
    <!-- Top controls (you can add your header here if needed) -->

    <!-- Main price chart (candlesticks) -->
    <section style="margin-top: 12px">
      <CandleChart />
      <RSIChart />
    </section>

    <!-- You can render RSI/MACD/Bollinger/Momentum below when ready -->
  </main>
</template>
