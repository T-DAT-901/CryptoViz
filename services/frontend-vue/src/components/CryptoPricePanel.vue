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
    <div class="crypto-header">
      <div class="crypto-icon">
        <Bitcoin class="bitcoin-icon" />
        <div class="crypto-info">
          <h1 class="crypto-name">{{ cryptoName || "Bitcoin" }}</h1>
          <span class="crypto-symbol">{{ symbol || "BTC" }}</span>
          <span class="crypto-rank">#1</span>
        </div>
      </div>

      <!-- Actions boutons -->
      <div class="action-buttons">
        <button class="action-btn favorite-btn">
          <Star class="action-icon" />
        </button>
        <button class="action-btn share-btn">
          <Share2 class="action-icon" />
        </button>
        <button class="action-btn insights-btn">
          <Activity class="action-icon" />
        </button>
      </div>
    </div>

    <!-- Prix principal -->
    <div class="main-price">
      <h2 class="price-value">
        {{
          currentPrice.toLocaleString("fr-FR", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
          })
        }}
        €
      </h2>
      <div
        class="price-change"
        :class="{ positive: isPositive, negative: !isPositive }"
      >
        <TrendingUp v-if="isPositive" class="change-icon" />
        <TrendingDown v-else class="change-icon" />
        <span class="change-text">
          {{ isPositive ? "+" : "" }}{{ priceChangePercent.toFixed(2) }}% (1Y)
        </span>
      </div>
    </div>

    <!-- Convertisseur BTC/EUR -->
    <div class="converter-section">
      <h3 class="section-title">Convertisseur BTC / EUR</h3>
      <div class="converter">
        <div class="currency-input">
          <label class="currency-label">BTC</label>
          <input
            v-model.number="btcAmount"
            @input="updateEurFromBtc"
            type="number"
            step="0.00000001"
            class="amount-input"
          />
        </div>
        <div class="currency-input">
          <label class="currency-label">EUR</label>
          <input
            v-model.number="eurAmount"
            @input="updateBtcFromEur"
            type="number"
            step="0.01"
            class="amount-input"
          />
        </div>
      </div>
    </div>

    <!-- Performance de prix -->
    <div class="performance-section">
      <div class="section-header">
        <h3 class="section-title">Performance de prix</h3>
        <select class="timeframe-select">
          <option value="24h">24h</option>
          <option value="7d">7d</option>
          <option value="1M">1M</option>
          <option value="1Y">1Y</option>
        </select>
      </div>

      <div class="performance-stats">
        <div class="stat-row">
          <span class="stat-label">Bas</span>
          <span class="stat-value"
            >€{{
              low24h.toLocaleString("fr-FR", { minimumFractionDigits: 2 })
            }}</span
          >
        </div>
        <div class="stat-row">
          <span class="stat-label">Haut</span>
          <span class="stat-value"
            >€{{
              high24h.toLocaleString("fr-FR", { minimumFractionDigits: 2 })
            }}</span
          >
        </div>
      </div>

      <!-- Barre de progression -->
      <div class="price-range-bar">
        <div class="range-track">
          <div
            class="range-progress"
            :style="{
              width: ((currentPrice - low24h) / (high24h - low24h)) * 100 + '%',
            }"
          ></div>
        </div>
      </div>
    </div>

    <!-- Historiques -->
    <div class="historical-section">
      <div class="historical-item">
        <div class="historical-main">
          <span class="historical-label">Haut historique</span>
          <span class="historical-price">€109,105.05</span>
        </div>
        <div class="historical-meta">
          <span class="historical-date">oct. 06, 2025 (il y a 25 jours)</span>
          <span class="historical-change negative">-12.92%</span>
        </div>
      </div>

      <div class="historical-item">
        <div class="historical-main">
          <span class="historical-label">Bas historique</span>
          <span class="historical-price">€0.04206</span>
        </div>
        <div class="historical-meta">
          <span class="historical-date">juil. 14, 2010 (il y a 15 ans)</span>
          <span class="historical-change positive">+225896044.65%</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.crypto-price-panel {
  background: #040d12;
  border-radius: 12px;
  padding: 16px;
  color: #e5e7eb;
  border: 1px solid #1a1a2e;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
  width: 100%;
  max-width: 100%;
  height: fit-content;
  box-sizing: border-box;
}

/* Header */
.crypto-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 20px;
}

.crypto-icon {
  display: flex;
  align-items: center;
  gap: 10px;
}

.bitcoin-icon {
  width: 24px;
  height: 24px;
  color: #f7931a;
  stroke-width: 2;
}

.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: #1a1a2e;
  color: #9ca3af;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: #2d2d47;
  color: #e5e7eb;
  transform: translateY(-1px);
}

.action-icon {
  width: 16px;
  height: 16px;
  stroke-width: 2;
}

.crypto-info {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.crypto-name {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  color: #e5e7eb;
}

.crypto-symbol {
  font-size: 12px;
  color: #9ca3af;
  font-weight: 500;
}

.crypto-rank {
  font-size: 10px;
  background: #1a1a2e;
  color: #9ca3af;
  padding: 2px 6px;
  border-radius: 8px;
  width: fit-content;
}

.favorite-actions {
  display: flex;
  gap: 6px;
}

.favorite-btn,
.share-btn {
  background: rgba(26, 26, 46, 0.8);
  border: 1px solid #374151;
  border-radius: 6px;
  padding: 6px 8px;
  color: #9ca3af;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 3px;
  font-size: 10px;
}

.favorite-btn:hover,
.share-btn:hover {
  background: #374151;
  color: #e5e7eb;
}

/* Prix principal */
.main-price {
  margin-bottom: 16px;
}

.price-value {
  font-size: 28px;
  font-weight: 700;
  margin: 0 0 6px 0;
  color: #e5e7eb;
  letter-spacing: -1px;
  line-height: 1.1;
}

.price-change {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
}

.price-change.positive {
  color: #10b981;
}

.price-change.negative {
  color: #ef4444;
}

.change-icon {
  width: 16px;
  height: 16px;
  stroke-width: 2;
}

/* Insight */
.price-insight {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(79, 70, 229, 0.1);
  border: 1px solid rgba(79, 70, 229, 0.3);
  border-radius: 10px;
  padding: 10px 12px;
  margin-bottom: 20px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.price-insight:hover {
  background: rgba(79, 70, 229, 0.15);
  border-color: rgba(79, 70, 229, 0.4);
}

.insight-icon {
  font-size: 16px;
}

.insight-text {
  flex: 1;
  font-size: 13px;
  color: #e5e7eb;
}

.insight-arrow {
  background: none;
  border: none;
  color: #9ca3af;
  font-size: 16px;
  cursor: pointer;
}

/* Sections */
.section-title {
  font-size: 14px;
  font-weight: 600;
  margin: 0 0 12px 0;
  color: #e5e7eb;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.timeframe-select {
  background: #1a1a2e;
  border: 1px solid #374151;
  border-radius: 6px;
  color: #e5e7eb;
  padding: 4px 8px;
  font-size: 12px;
}

/* Convertisseur */
.converter-section {
  margin-bottom: 20px;
}

.converter {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.currency-input {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #1a1a2e;
  border: 1px solid #374151;
  border-radius: 6px;
  padding: 10px 12px;
}

.currency-label {
  font-size: 12px;
  font-weight: 500;
  color: #9ca3af;
}

.amount-input {
  background: none;
  border: none;
  color: #e5e7eb;
  font-size: 14px;
  font-weight: 600;
  text-align: right;
  width: 60%;
  outline: none;
}

.amount-input::-webkit-outer-spin-button,
.amount-input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.amount-input[type="number"] {
  appearance: textfield;
  -moz-appearance: textfield;
}

/* Performance */
.performance-section {
  margin-bottom: 20px;
}

.performance-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
}

.stat-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-label {
  font-size: 12px;
  color: #9ca3af;
}

.stat-value {
  font-size: 12px;
  font-weight: 600;
  color: #e5e7eb;
}

.price-range-bar {
  margin-top: 12px;
}

.range-track {
  width: 100%;
  height: 4px;
  background: #374151;
  border-radius: 2px;
  overflow: hidden;
}

.range-progress {
  height: 100%;
  background: linear-gradient(90deg, #10b981, #4f46e5);
  transition: width 0.3s ease;
}

/* Historiques */
.historical-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.historical-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.historical-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.historical-label {
  font-size: 12px;
  color: #e5e7eb;
  font-weight: 500;
}

.historical-price {
  font-size: 12px;
  font-weight: 600;
  color: #e5e7eb;
}

.historical-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.historical-date {
  font-size: 10px;
  color: #6b7280;
}

.historical-change {
  font-size: 10px;
  font-weight: 500;
}

.historical-change.positive {
  color: #10b981;
}

.historical-change.negative {
  color: #ef4444;
}

/* Responsive */
@media (max-width: 768px) {
  .crypto-price-panel {
    padding: 12px;
  }

  .price-value {
    font-size: 24px;
  }

  .crypto-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }
}
</style>
