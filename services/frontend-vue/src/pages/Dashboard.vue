<script setup lang="ts">
import IndicatorsPanel from "@/components/IndicatorsPanel.vue";
import RSIChart from "@/components/charts/RSIChart.vue";
import MACDChart from "@/components/charts/MACDChart.vue";
import BollingerChart from "@/components/charts/BollingerChart.vue";
import MomentumChart from "@/components/charts/MomentumChart.vue";
import { useIndicatorsStore } from "@/stores/indicators";
const ind = useIndicatorsStore();
</script>

<template>
  <!-- ... header + interval selector + price charts ... -->
  <IndicatorsPanel />

  <section style="display: grid; gap: 16px; margin: 12px 0">
    <RSIChart v-if="ind.showRSI" symbol="BTC" />
    <MACDChart v-if="ind.showMACD" symbol="BTC" />
    <BollingerChart v-if="ind.showBollinger" symbol="BTC" />
    <MomentumChart v-if="ind.showMomentum" symbol="BTC" />
  </section>
</template>
