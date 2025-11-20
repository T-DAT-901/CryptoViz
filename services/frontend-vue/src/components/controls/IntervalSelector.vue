<script setup lang="ts">
import { Clock } from "lucide-vue-next";
import { useMarketStore } from "@/stores/market";
import { SUPPORTED_INTERVALS } from "@/constants/intervals";

const marketStore = useMarketStore();

const selectInterval = (interval: string) => {
  marketStore.switchInterval(interval);
};
</script>

<template>
  <div class="interval-selector">
    <div class="interval-selector__header">
      <Clock class="interval-selector__icon" />
      <span class="interval-selector__title">Interval</span>
    </div>
    <div class="interval-selector__buttons">
      <button
        v-for="interval in SUPPORTED_INTERVALS"
        :key="interval.value"
        class="interval-selector__button"
        :class="{
          'interval-selector__button--active':
            marketStore.interval === interval.value,
        }"
        @click="selectInterval(interval.value)"
        :title="interval.description"
      >
        {{ interval.label }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.interval-selector {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.interval-selector__header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #94a3b8;
  font-size: 0.875rem;
  font-weight: 500;
}

.interval-selector__icon {
  width: 1rem;
  height: 1rem;
}

.interval-selector__title {
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.interval-selector__buttons {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.interval-selector__button {
  padding: 0.5rem 1rem;
  background: #1e293b;
  border: 1px solid #334155;
  border-radius: 0.375rem;
  color: #94a3b8;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.interval-selector__button:hover {
  background: #334155;
  border-color: #475569;
  color: #cbd5e1;
}

.interval-selector__button--active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #ffffff;
}

.interval-selector__button--active:hover {
  background: #2563eb;
  border-color: #2563eb;
}
</style>
