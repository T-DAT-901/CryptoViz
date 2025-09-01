<script setup lang="ts">
import { useIndicatorsStore } from "@/stores/indicators";
const ind = useIndicatorsStore();
</script>

<template>
  <div style="display: flex; gap: 16px; flex-wrap: wrap; margin: 8px 0">
    <label
      ><input type="checkbox" v-model="ind.showRSI" /> RSI ({{
        ind.rsiPeriod
      }})</label
    >
    <label
      ><input type="checkbox" v-model="ind.showMACD" /> MACD ({{
        ind.macdFast
      }},{{ ind.macdSlow }},{{ ind.macdSignal }})</label
    >
    <label
      ><input type="checkbox" v-model="ind.showBollinger" /> Bollinger ({{
        ind.bbPeriod
      }},{{ ind.bbStd }})</label
    >
    <label
      ><input type="checkbox" v-model="ind.showMomentum" /> Momentum ({{
        ind.momPeriod
      }})</label
    >
  </div>
</template>
