<!-- Graph bandes de Bollinger -->
<script setup lang="ts">
defineProps<{ symbol: string }>();
</script>

<template>
  <div class="bollinger-chart">
    <div class="bollinger-chart__header">
      <h3 class="bollinger-chart__title">Bollinger Bands</h3>
      <span class="bollinger-chart__symbol">{{ symbol }}</span>
    </div>
    <div class="bollinger-chart__content">
      <div class="bollinger-chart__placeholder">
        Graphique Bollinger Bands - À implémenter
      </div>
    </div>
  </div>
</template>
