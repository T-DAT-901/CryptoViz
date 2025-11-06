<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowLeft } from "lucide-vue-next";
import TradingChart from "@/components/charts/TradingChart.vue";
import CryptoPricePanel from "@/components/CryptoPricePanel.vue";
import CommunitySentiment from "@/components/CommunitySentiment.vue";
import IndicatorsPanel from "@/components/IndicatorsPanel.vue";
import ViewModeToggle from "@/components/controls/ViewModeToggle.vue";
import RSIChart from "@/components/charts/RSIChart.vue";
import MACDChart from "@/components/charts/MACDChart.vue";
import BollingerChart from "@/components/charts/BollingerChart.vue";
import MomentumChart from "@/components/charts/MomentumChart.vue";
import NewsFeed from "@/components/NewsFeed.vue";
import { useIndicatorsStore } from "@/stores/indicators";

const route = useRoute();
const router = useRouter();
const symbol = computed(() => (route.params.symbol as string) || "btc");
const symbolPair = computed(() => symbol.value.toUpperCase() + "USDT");

const indicatorsStore = useIndicatorsStore();
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
    <header class="dashboard-header">
      <div class="dashboard-header__left">
        <button class="dashboard-back-btn" @click="goBack">
          <ArrowLeft class="dashboard-back-icon" />
          <span>Retour aux cryptos</span>
        </button>
        <h1 class="dashboard-title">{{ cryptoName }} Dashboard</h1>
      </div>
      <ViewModeToggle />
    </header>

    <main class="dashboard-grid">
      <aside class="dashboard-column dashboard-column--left">
        <div class="dashboard-stack">
          <CryptoPricePanel :symbol="symbolPair" :crypto-name="cryptoName" />
        </div>
      </aside>

      <section class="dashboard-column dashboard-column--center">
        <div class="dashboard-stack">
          <!-- Compact mode: unified chart + controls -->
          <template v-if="indicatorsStore.layoutMode === 'compact'">
            <TradingChart />
            <IndicatorsPanel />
          </template>

          <!-- Detailed mode: separate charts -->
          <template v-else>
            <TradingChart />
            <RSIChart v-if="indicatorsStore.showRSI" :symbol="symbolPair" />
            <MACDChart v-if="indicatorsStore.showMACD" :symbol="symbolPair" />
            <BollingerChart
              v-if="indicatorsStore.showBollinger"
              :symbol="symbolPair"
            />
            <MomentumChart
              v-if="indicatorsStore.showMomentum"
              :symbol="symbolPair"
            />
            <IndicatorsPanel />
          </template>
        </div>
      </section>

      <aside class="dashboard-column dashboard-column--right">
        <div class="dashboard-stack">
          <CommunitySentiment symbol="BTCUSDT" />
          <NewsFeed />
        </div>
      </aside>
    </main>
  </div>
</template>
