<script setup lang="ts">
import CandleChart from "@/components/charts/CandleChart.vue";
import RSIChart from "@/components/charts/RSIChart.vue";
</script>

<template>
  <main class="dash">
    <!-- Colonne gauche (placeholder) -->
    <aside class="col left"></aside>

    <!-- Centre (graphiques) -->
    <section class="col center">
      <div class="stack">
        <CandleChart />
        <RSIChart />
      </div>
    </section>

    <!-- Colonne droite (placeholder) -->
    <aside class="col right"></aside>
  </main>
</template>

<style scoped>
.dash {
  display: grid;
  grid-template-columns: 360px 1fr 420px; /* proches de ta maquette */
  gap: 0;
  height: 100vh; /* plein écran */
  background: #040d12; /* même fond que global */
}

/* chaque colonne scrolle seule */
.col {
  padding: 12px;
  min-height: 0; /* nécessaire pour que overflow-y fonctionne en grid */
  overflow-y: auto;
}

/* lignes de séparation noires */
.left {
  border-right: 0.5px solid #93b1a6;
}
.right {
  border-left: 0.5px solid #93b1a6;
}

.stack {
  display: grid;
  gap: 12px;
}
</style>
