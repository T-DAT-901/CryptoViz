<script setup lang="ts">
import { useIndicatorsStore } from "@/stores/indicators";
import { TrendingUp, BarChart, Activity, Zap } from "lucide-vue-next";
const ind = useIndicatorsStore();
</script>

<template>
  <div class="indicators-panel">
    <label class="indicators-panel-item">
      <input type="checkbox" v-model="ind.showRSI" />
      <TrendingUp class="indicators-panel-icon" />
      RSI ({{ ind.rsiPeriod }})
    </label>
    <label class="indicators-panel-item">
      <input type="checkbox" v-model="ind.showMACD" />
      <BarChart class="indicators-panel-icon" />
      MACD ({{ ind.macdFast }},{{ ind.macdSlow }},{{ ind.macdSignal }})
    </label>
    <label class="indicators-panel-item">
      <input type="checkbox" v-model="ind.showBollinger" />
      <Activity class="indicators-panel-icon" />
      Bollinger ({{ ind.bbPeriod }},{{ ind.bbStd }})
    </label>
    <label class="indicators-panel-item">
      <input type="checkbox" v-model="ind.showMomentum" />
      <Zap class="indicators-panel-icon" />
      Momentum ({{ ind.momPeriod }})
    </label>
  </div>
</template>
