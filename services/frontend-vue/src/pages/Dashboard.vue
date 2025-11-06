<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowLeft } from "lucide-vue-next";
import TradingChart from "@/components/charts/TradingChart.vue";
import CryptoPricePanel from "@/components/CryptoPricePanel.vue";
import CommunitySentiment from "@/components/CommunitySentiment.vue";
import IndicatorsPanel from "@/components/IndicatorsPanel.vue";
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
      <button class="dashboard-back-btn" @click="goBack">
        <ArrowLeft class="dashboard-back-icon" />
        <span>Retour aux cryptos</span>
      </button>
      <h1 class="dashboard-title">{{ cryptoName }} Dashboard</h1>
    </header>

    <main class="dashboard-grid">
      <!-- Colonne gauche : infos crypto -->
      <aside class="dashboard-column dashboard-column--left">
        <div class="dashboard-stack">
          <CryptoPricePanel :symbol="symbolPair" :crypto-name="cryptoName" />
        </div>
      </aside>

      <!-- Centre (graphiques) -->
      <section class="dashboard-column dashboard-column--center">
        <div class="dashboard-stack">
          <TradingChart />
          <IndicatorsPanel />
          <!-- <RSIChart /> Temporairement désactivé pour tester -->
        </div>
      </section>

      <!-- Colonne droite : sentiment + news -->
      <aside class="dashboard-column dashboard-column--right">
        <div class="dashboard-stack">
          <CommunitySentiment symbol="BTCUSDT" />
          <NewsFeed />
        </div>
      </aside>
    </main>
  </div>
</template>
