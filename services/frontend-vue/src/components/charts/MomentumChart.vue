<script setup lang="ts">
defineProps<{ symbol: string }>();
</script>

<template>
  <div class="momentum-chart">
    <div class="momentum-chart__header">
      <h3 class="momentum-chart__title">Momentum</h3>
      <span class="momentum-chart__symbol">{{ symbol }}</span>
    </div>
    <div class="momentum-chart__content">
      <div class="momentum-chart__placeholder">
        Graphique Momentum - À implémenter
      </div>
    </div>
  </div>
</template>
