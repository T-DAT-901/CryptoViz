<!-- Graph MACD -->
<script setup lang="ts">
defineProps<{ symbol: string }>();
</script>

<template>
  <div class="macd-chart">
    <div class="macd-chart__header">
      <h3 class="macd-chart__title">MACD</h3>
      <span class="macd-chart__symbol">{{ symbol }}</span>
    </div>
    <div class="macd-chart__content">
      <div class="macd-chart__placeholder">Graphique MACD - À implémenter</div>
    </div>
  </div>
</template>
