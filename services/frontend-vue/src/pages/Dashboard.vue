<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowLeft } from "lucide-vue-next";
import TradingChart from "@/components/charts/TradingChart.vue";
import CryptoPricePanel from "@/components/CryptoPricePanel.vue";
import CommunitySentiment from "@/components/CommunitySentiment.vue";
// import RSIChart from "@/components/charts/RSIChart.vue"; // Temporairement désactivé pour tester
import NewsFeed from "@/components/NewsFeed.vue";

// Récupération du symbole depuis l'URL
const route = useRoute();
const router = useRouter();
const symbol = computed(() => (route.params.symbol as string) || "btc");
const symbolPair = computed(() => symbol.value.toUpperCase() + "USDT");

// Mapping des noms de cryptos
const cryptoNames: Record<string, string> = {
  btc: "Bitcoin",
  eth: "Ethereum",
  usdt: "Tether",
  xrp: "XRP",
  bnb: "BNB",
  sol: "Solana",
  usdc: "USD Coin",
  trx: "TRON",
  doge: "Dogecoin",
};

const cryptoName = computed(
  () => cryptoNames[symbol.value] || symbol.value.toUpperCase()
);

const goBack = () => {
  router.push("/");
};
</script>

<template>
  <div class="dashboard-page">
    <!-- Header avec navigation -->
    <header class="dashboard-header">
      <button class="back-btn" @click="goBack">
        <ArrowLeft class="back-icon" />
        <span>Retour aux cryptos</span>
      </button>
      <h1 class="crypto-title">{{ cryptoName }} Dashboard</h1>
    </header>

    <main class="dash">
      <!-- Colonne gauche : infos crypto -->
      <aside class="col left">
        <div class="stack">
          <CryptoPricePanel :symbol="symbolPair" :crypto-name="cryptoName" />
        </div>
      </aside>

      <!-- Centre (graphiques) -->
      <section class="col center">
        <div class="stack">
          <TradingChart />
          <!-- <RSIChart /> Temporairement désactivé pour tester -->
        </div>
      </section>

      <!-- Colonne droite : sentiment + news -->
      <aside class="col right">
        <div class="stack">
          <CommunitySentiment symbol="BTCUSDT" />
          <NewsFeed />
        </div>
      </aside>
    </main>
  </div>
</template>
<style scoped>
.dashboard-page {
  min-height: 100vh;
  background: #040d12;
}

.dashboard-header {
  background: #0b0e11;
  border-bottom: 1px solid #1a1f24;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  gap: 20px;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #1a1a2e;
  border: 1px solid #2d2d44;
  color: #9ca3af;
  padding: 8px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 14px;
}

.back-btn:hover {
  background: #2d2d47;
  color: #e5e7eb;
}

.back-icon {
  width: 16px;
  height: 16px;
  stroke-width: 2;
}

.crypto-title {
  color: #e5e7eb;
  font-size: 24px;
  font-weight: 600;
  margin: 0;
}

.dash {
  display: grid;
  grid-template-columns: 360px 1fr 420px;
  gap: 0;
  height: calc(100vh - 73px); /* Ajustement pour le header */
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
