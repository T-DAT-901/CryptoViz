<script setup lang="ts">
import CandleChart from "@/components/charts/CandleChart.vue";
import RSIChart from "@/components/charts/RSIChart.vue";
import NewsFeed from "@/components/NewsFeed.vue";
</script>

<template>
  <main class="dash">
    <!-- Colonne gauche : infos crypto -->
    <aside class="col left"></aside>

    <!-- Centre (graphiques) -->
    <section class="col center">
      <div class="stack">
        <CandleChart />
        <RSIChart />
      </div>
    </section>

    <!-- Colonne droite : news -->
    <aside class="col right">
      <div class="stack">
        <NewsFeed />
      </div>
    </aside>
  </main>
</template>

<style scoped>
.dash {
  display: grid;
  grid-template-columns: 360px 1fr 420px;
  gap: 0;
  height: 100vh;
  background: #040d12;
}

.col {
  padding: 12px;
  min-height: 0;
  overflow-y: auto;
}

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
