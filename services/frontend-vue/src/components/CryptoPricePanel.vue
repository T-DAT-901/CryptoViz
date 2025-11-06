<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useTradingWebSocket, useLivePrices } from "@/services/websocket";
import {
  Bitcoin,
  TrendingUp,
  TrendingDown,
  Star,
  Share2,
  Activity,
} from "lucide-vue-next";

const props = defineProps<{
  symbol?: string;
  cryptoName?: string;
}>();

// WebSocket pour prix en temps réel
const { connect, disconnect, isConnected } = useTradingWebSocket();
const { priceData: livePrice } = useLivePrices(props.symbol || "BTCUSDT");

// État local pour le convertisseur
const btcAmount = ref(1);
const eurAmount = ref(94989.59);

// Prix et données calculées
const currentPrice = computed(() => {
  if (livePrice.value) {
    return livePrice.value.price;
  }
  return 70337.27; // Prix de fallback selon l'image
});

const priceChange = computed(() => {
  if (livePrice.value) {
    return livePrice.value.change;
  }
  return 99.53; // Change de fallback
});

const priceChangePercent = computed(() => {
  if (livePrice.value) {
    return livePrice.value.changePercent;
  }
  return 0.14; // Pourcentage de fallback
});

const isPositive = computed(() => priceChange.value > 0);

const high24h = computed(() => {
  if (livePrice.value) {
    return livePrice.value.high24h;
  }
  return 95422.12; // Haut 24h de fallback
});

const low24h = computed(() => {
  if (livePrice.value) {
    return livePrice.value.low24h;
  }
  return 91968.39; // Bas 24h de fallback
});

// Mise à jour du convertisseur
function updateEurFromBtc() {
  eurAmount.value = btcAmount.value * currentPrice.value;
}

function updateBtcFromEur() {
  btcAmount.value = eurAmount.value / currentPrice.value;
}

// Lifecycle
onMounted(async () => {
  try {
    await connect();
  } catch (error) {
    console.warn("WebSocket connection failed for price panel");
  }
});

onUnmounted(() => {
  disconnect();
});
</script>

<template>
  <div class="crypto-price-panel">
    <!-- Header avec nom et prix principal -->
    <div class="crypto-price-panel-header">
      <div class="crypto-price-panel-icon-section">
        <Bitcoin class="crypto-price-panel-bitcoin-icon" />
        <div class="crypto-price-panel-info">
          <h1 class="crypto-price-panel-name">{{ cryptoName || "Bitcoin" }}</h1>
          <span class="crypto-price-panel-symbol">{{ symbol || "BTC" }}</span>
          <span class="crypto-price-panel-rank">#1</span>
        </div>
      </div>

      <!-- Actions boutons -->
      <div class="crypto-price-panel-actions">
        <button class="crypto-price-panel-action-btn">
          <Star class="crypto-price-panel-action-icon" />
        </button>
        <button class="crypto-price-panel-action-btn">
          <Share2 class="crypto-price-panel-action-icon" />
        </button>
        <button class="crypto-price-panel-action-btn">
          <Activity class="crypto-price-panel-action-icon" />
        </button>
      </div>
    </div>

    <!-- Prix principal -->
    <div class="crypto-price-panel-main-price">
      <h2 class="crypto-price-panel-price-value">
        {{
          currentPrice.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          })
        }}
        €
      </h2>
      <div
        class="crypto-price-panel-price-change"
        :class="{
          'crypto-price-panel-price-change--positive': isPositive,
          'crypto-price-panel-price-change--negative': !isPositive,
        }"
      >
        <TrendingUp v-if="isPositive" class="crypto-price-panel-change-icon" />
        <TrendingDown v-else class="crypto-price-panel-change-icon" />
        <span class="change-text">
          {{ isPositive ? "+" : "" }}{{ priceChangePercent.toFixed(2) }}% (1Y)
        </span>
      </div>
    </div>

    <!-- Convertisseur BTC/EUR -->
    <div class="crypto-price-panel-converter-section">
      <h3 class="crypto-price-panel-section-title">Convertisseur BTC / EUR</h3>
      <div class="crypto-price-panel-converter">
        <div class="crypto-price-panel-currency-input">
          <label class="crypto-price-panel-currency-label">BTC</label>
          <input
            v-model.number="btcAmount"
            @input="updateEurFromBtc"
            type="number"
            step="0.00000001"
            class="crypto-price-panel-amount-input"
          />
        </div>
        <div class="crypto-price-panel-currency-input">
          <label class="crypto-price-panel-currency-label">EUR</label>
          <input
            v-model.number="eurAmount"
            @input="updateBtcFromEur"
            type="number"
            step="0.01"
            class="crypto-price-panel-amount-input"
          />
        </div>
      </div>
    </div>

    <!-- Performance de prix -->
    <div class="crypto-price-panel-performance-section">
      <div class="crypto-price-panel-section-header">
        <h3 class="crypto-price-panel-section-title">Performance de prix</h3>
        <select class="crypto-price-panel-timeframe-select">
          <option value="24h">24h</option>
          <option value="7d">7d</option>
          <option value="1M">1M</option>
          <option value="1Y">1Y</option>
        </select>
      </div>

      <div class="crypto-price-panel-performance-stats">
        <div class="crypto-price-panel-stat-row">
          <span class="crypto-price-panel-stat-label">Bas</span>
          <span class="crypto-price-panel-stat-value"
            >€{{
              low24h.toLocaleString("fr-FR", { minimumFractionDigits: 2 })
            }}</span
          >
        </div>
        <div class="crypto-price-panel-stat-row">
          <span class="crypto-price-panel-stat-label">Haut</span>
          <span class="crypto-price-panel-stat-value"
            >€{{
              high24h.toLocaleString("fr-FR", { minimumFractionDigits: 2 })
            }}</span
          >
        </div>
      </div>

      <!-- Barre de progression -->
      <div class="crypto-price-panel-price-range-bar">
        <div class="crypto-price-panel-range-track">
          <div
            class="crypto-price-panel-range-progress"
            :style="{
              width: ((currentPrice - low24h) / (high24h - low24h)) * 100 + '%',
            }"
          ></div>
        </div>
      </div>
    </div>

    <!-- Historiques -->
    <div class="crypto-price-panel-historical-section">
      <div class="crypto-price-panel-historical-item">
        <div class="crypto-price-panel-historical-main">
          <span class="crypto-price-panel-historical-label"
            >Haut historique</span
          >
          <span class="crypto-price-panel-historical-price">€109,105.05</span>
        </div>
        <div class="crypto-price-panel-historical-meta">
          <span class="crypto-price-panel-historical-date"
            >oct. 06, 2025 (il y a 25 jours)</span
          >
          <span
            class="crypto-price-panel-historical-change crypto-price-panel-historical-change--negative"
            >-12.92%</span
          >
        </div>
      </div>

      <div class="crypto-price-panel-historical-item">
        <div class="crypto-price-panel-historical-main">
          <span class="crypto-price-panel-historical-label"
            >Bas historique</span
          >
          <span class="crypto-price-panel-historical-price">€0.04206</span>
        </div>
        <div class="crypto-price-panel-historical-meta">
          <span class="crypto-price-panel-historical-date"
            >juil. 14, 2010 (il y a 15 ans)</span
          >
          <span
            class="crypto-price-panel-historical-change crypto-price-panel-historical-change--positive"
            >+225896044.65%</span
          >
        </div>
      </div>
    </div>
  </div>
</template>
