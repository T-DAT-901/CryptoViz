<script setup lang="ts">
import TradingChart from "@/components/charts/TradingChart.vue";
// import RSIChart from "@/components/charts/RSIChart.vue"; // Temporairement désactivé pour tester
import NewsFeed from "@/components/NewsFeed.vue";
</script>

<template>
  <main class="dash">
    <!-- Colonne gauche : infos crypto -->
    <aside class="col left"></aside>

    <!-- Centre (graphiques) -->
    <section class="col center">
      <div class="stack">
        <TradingChart />
        <!-- <RSIChart /> Temporairement désactivé pour tester -->
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

@media (max-width: 900px) {
  .dash {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
    height: auto;
    min-height: 100vh;
    background: #040d12;
    padding: 0;
  }
}

@media (max-width: 900px) {
  .dash {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
    height: auto;
    min-height: 100vh;
  }
  .col.left,
  .col.right {
    border: none;
    padding: 0 0 12px 0;
  }
  .col.center {
    padding: 0;
  }
  .stack {
    gap: 20px;
  }
}

.col {
  padding: 16px 0;
  min-height: 0;
  overflow-y: auto;
}

@media (max-width: 900px) {
  .col {
    padding: 0 0 0 0;
    min-width: 0;
    overflow: visible;
  }
}

@media (max-width: 900px) {
  .col {
    padding: 8px 4vw;
    overflow: visible;
  }
}

.left {
  border-right: 0.5px solid #93b1a6;
}
.right {
  border-left: 0.5px solid #93b1a6;
}

@media (max-width: 900px) {
  .left,
  .right {
    border: none;
  }
}

.stack {
  display: grid;
  gap: 18px;
  margin: 0 16px;
}

@media (max-width: 900px) {
  .stack {
    gap: 24px;
    margin: 0 8px;
  }
}
</style>
