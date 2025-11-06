<script setup lang="ts">
import { useIndicatorsStore } from "@/stores/indicators";
import { TrendingUp, BarChart, Activity, Zap } from "lucide-vue-next";
const ind = useIndicatorsStore();
</script>

<template>
  <div class="indicators-panel">
    <label class="indicator-item">
      <input type="checkbox" v-model="ind.showRSI" />
      <TrendingUp class="indicator-icon" />
      RSI ({{ ind.rsiPeriod }})
    </label>
    <label class="indicator-item">
      <input type="checkbox" v-model="ind.showMACD" />
      <BarChart class="indicator-icon" />
      MACD ({{ ind.macdFast }},{{ ind.macdSlow }},{{ ind.macdSignal }})
    </label>
    <label class="indicator-item">
      <input type="checkbox" v-model="ind.showBollinger" />
      <Activity class="indicator-icon" />
      Bollinger ({{ ind.bbPeriod }},{{ ind.bbStd }})
    </label>
    <label class="indicator-item">
      <input type="checkbox" v-model="ind.showMomentum" />
      <Zap class="indicator-icon" />
      Momentum ({{ ind.momPeriod }})
    </label>
  </div>
</template>

<style scoped>
.indicators-panel {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  margin: 8px 0;
}

.indicator-item {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  padding: 6px 10px;
  border-radius: 6px;
  background: #0b0e11;
  border: 1px solid #1a1f24;
  color: #e5e7eb;
  font-size: 14px;
  transition: all 0.2s ease;
}

.indicator-item:hover {
  background: #141a1f;
  border-color: #2d2d44;
}

.indicator-icon {
  width: 14px;
  height: 14px;
  stroke-width: 2;
  color: #9ca3af;
}

.indicator-item input[type="checkbox"] {
  margin-right: 2px;
}
</style>
