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

// WebSocket connection for real-time prices
const { connect, disconnect, isConnected } = useTradingWebSocket();
const { priceData: livePrice } = useLivePrices(props.symbol || "BTCUSDT");

// Converter state
const btcAmount = ref(1);
const eurAmount = ref(94989.59);

// Current prices with fallback if WebSocket unavailable
const currentPrice = computed(() => {
  return livePrice.value?.price || 70337.27;
});

const priceChange = computed(() => {
  return livePrice.value?.change || 99.53;
});

const priceChangePercent = computed(() => {
  return livePrice.value?.changePercent || 0.14;
});

const isPositive = computed(() => priceChange.value > 0);

// Price formatting for display
const formattedPrice = computed(() => {
  return currentPrice.value.toLocaleString("fr-FR", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
});

// High/low data with fallback
const high24h = computed(() => livePrice.value?.high24h || 95422.12);
const low24h = computed(() => livePrice.value?.low24h || 91968.39);

const formattedHigh = computed(() => {
  return high24h.value.toLocaleString("fr-FR", { minimumFractionDigits: 2 });
});

const formattedLow = computed(() => {
  return low24h.value.toLocaleString("fr-FR", { minimumFractionDigits: 2 });
});

// CSS class for price change
const priceChangeClass = computed(() => {
  return isPositive.value
    ? "crypto-price-change--positive"
    : "crypto-price-change--negative";
});

// Progress bar width calculation
const progressWidth = computed(() => {
  const range = high24h.value - low24h.value;
  const position = currentPrice.value - low24h.value;
  return `${(position / range) * 100}%`;
});

// Converter update functions
function updateEurFromBtc() {
  eurAmount.value = btcAmount.value * currentPrice.value;
}

function updateBtcFromEur() {
  btcAmount.value = eurAmount.value / currentPrice.value;
}

// Initialize WebSocket on mount
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
  <div class="crypto-card">
    <!-- Crypto header -->
    <div class="crypto-header">
      <div class="crypto-info">
        <Bitcoin class="crypto-icon" />
        <div class="crypto-details">
          <h1 class="crypto-name">{{ cryptoName || "Bitcoin" }}</h1>
          <span class="crypto-symbol">{{ symbol || "BTC" }}</span>
          <span class="crypto-rank">#1</span>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="crypto-actions">
        <button class="crypto-action-btn">
          <Star class="crypto-action-icon" />
        </button>
        <button class="crypto-action-btn">
          <Share2 class="crypto-action-icon" />
        </button>
        <button class="crypto-action-btn">
          <Activity class="crypto-action-icon" />
        </button>
      </div>
    </div>

    <!-- Main price -->
    <div class="crypto-price-section">
      <h2 class="crypto-main-price">{{ formattedPrice }} €</h2>
      <div class="crypto-price-change" :class="priceChangeClass">
        <TrendingUp v-if="isPositive" class="crypto-trend-icon" />
        <TrendingDown v-else class="crypto-trend-icon" />
        <span
          >{{ isPositive ? "+" : "" }}{{ priceChangePercent.toFixed(2) }}%
          (1Y)</span
        >
      </div>
    </div>

    <!-- Converter -->
    <div class="crypto-converter">
      <h3 class="crypto-converter-title">Convertisseur BTC / EUR</h3>
      <div class="crypto-converter-inputs">
        <div class="crypto-input-group">
          <label class="crypto-input-label">BTC</label>
          <input
            v-model.number="btcAmount"
            @input="updateEurFromBtc"
            type="number"
            step="0.00000001"
            class="crypto-input-field"
          />
        </div>
        <div class="crypto-input-group">
          <label class="crypto-input-label">EUR</label>
          <input
            v-model.number="eurAmount"
            @input="updateBtcFromEur"
            type="number"
            step="0.01"
            class="crypto-input-field"
          />
        </div>
      </div>
    </div>

    <!-- Performance -->
    <div class="crypto-performance">
      <div class="crypto-performance-header">
        <h3 class="crypto-performance-title">Performance de prix</h3>
        <select class="crypto-timeframe-select">
          <option value="24h">24h</option>
          <option value="7d">7d</option>
          <option value="1M">1M</option>
          <option value="1Y">1Y</option>
        </select>
      </div>

      <div class="crypto-stats">
        <div class="crypto-stat">
          <span class="crypto-stat-label">Bas</span>
          <span class="crypto-stat-value">€{{ formattedLow }}</span>
        </div>
        <div class="crypto-stat">
          <span class="crypto-stat-label">Haut</span>
          <span class="crypto-stat-value">€{{ formattedHigh }}</span>
        </div>
      </div>

      <!-- Progress bar -->
      <div class="crypto-progress-bar">
        <div class="crypto-progress-track">
          <div
            class="crypto-progress-fill"
            :style="{ width: progressWidth }"
          ></div>
        </div>
      </div>
    </div>

    <!-- Historical data -->
    <div class="crypto-historical">
      <div class="crypto-historical-item">
        <div class="crypto-historical-main">
          <span class="crypto-historical-label">Haut historique</span>
          <span class="crypto-historical-price">€109,105.05</span>
        </div>
        <div class="crypto-historical-meta">
          <span class="crypto-historical-date"
            >oct. 06, 2025 (il y a 25 jours)</span
          >
          <span
            class="crypto-historical-change crypto-historical-change--negative"
            >-12.92%</span
          >
        </div>
      </div>

      <div class="crypto-historical-item">
        <div class="crypto-historical-main">
          <span class="crypto-historical-label">Bas historique</span>
          <span class="crypto-historical-price">€0.04206</span>
        </div>
        <div class="crypto-historical-meta">
          <span class="crypto-historical-date"
            >juil. 14, 2010 (il y a 15 ans)</span
          >
          <span
            class="crypto-historical-change crypto-historical-change--positive"
            >+225896044.65%</span
          >
        </div>
      </div>
    </div>
  </div>
</template>
