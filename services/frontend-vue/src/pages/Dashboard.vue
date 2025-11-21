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
const symbol = computed(() => {
  const encoded = (route.params.symbol as string) || "btc";
  // Décoder le symbole (ex: BTC%2FUSDT -> BTC/USDT)
  return decodeURIComponent(encoded).toUpperCase();
});
const symbolPair = computed(() => symbol.value);

const indicatorsStore = useIndicatorsStore();
const cryptoNames: Record<string, string> = {
  "BTC/USDT": "Bitcoin",
  "BTC/FDUSD": "Bitcoin",
  "ETH/USDT": "Ethereum",
  "ETH/FDUSD": "Ethereum",
  "USDT/USDT": "Tether",
  "XRP/USDT": "XRP",
  "XRP/FDUSD": "XRP",
  "BNB/USDT": "BNB",
  "BNB/FDUSD": "BNB",
  "SOL/USDT": "Solana",
  "SOL/FDUSD": "Solana",
  "USDC/USDT": "USD Coin",
  "USDC/FDUSD": "USD Coin",
  "TRX/USDT": "TRON",
  "TRX/FDUSD": "TRON",
  "DOGE/USDT": "Dogecoin",
  "DOGE/FDUSD": "Dogecoin",
  "ADA/USDT": "Cardano",
  "ADA/FDUSD": "Cardano",
  "DOT/USDT": "Polkadot",
  "DOT/FDUSD": "Polkadot",
  "LINK/USDT": "Chainlink",
  "LINK/FDUSD": "Chainlink",
  "ZEC/USDT": "Zcash",
  "ZEC/FDUSD": "Zcash",
};

const cryptoName = computed(() => {
  return cryptoNames[symbol.value] || symbol.value;
});

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
          <CryptoPricePanel
            :symbol="symbolPair"
            :crypto-name="cryptoName"
            :full-symbol="symbol"
          />
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
