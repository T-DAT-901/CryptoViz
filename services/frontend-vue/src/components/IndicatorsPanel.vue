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
  background: #070e10;
  border: 1px solid #1d2626;
  color: #e8f0f0;
  font-size: 14px;
  transition: all 0.2s ease;
}

.indicator-item:hover {
  background: #1d2626;
  border-color: #2d3e3e;
}

.indicator-icon {
  width: 14px;
  height: 14px;
  color: #7a9393;
}

.indicator-item input[type="checkbox"] {
  margin-right: 2px;
}
</style>
