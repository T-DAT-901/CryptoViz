<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { getRTClient } from "@/services/rt";
import {
  Bitcoin,
  TrendingUp,
  TrendingDown,
  ChevronDown,
} from "lucide-vue-next";
import { fetchCandles } from "@/services/markets.api";

const props = defineProps<{
  symbol?: string;
  cryptoName?: string;
  fullSymbol?: string; // Symbole complet avec slash (ex: BTC/USDT)
}>();

// État pour les prix en temps réel
const currentPrice = ref(0);
const high24h = ref(0);
const low24h = ref(0);
const previousPrice = ref(0);

// Timeframe pour la variation de prix
const selectedTimeframe = ref<"24h" | "7d" | "1M" | "1Y">("24h");
const timeframeOptions = ["24h", "7d", "1M", "1Y"] as const;
const isTimeframeDropdownOpen = ref(false);

// Closing prices par timeframe
const closingPrices = ref<Record<string, number>>({
  "24h": 0,
  "7d": 0,
  "1M": 0,
  "1Y": 0,
});

// État local pour le convertisseur
const btcAmount = ref(1);
const eurAmount = ref(0);

// Utiliser le fullSymbol ou fallback sur symbol
const realSymbol = computed(
  () => props.fullSymbol || props.symbol || "BTC/USDT"
);

// WebSocket real-time
const rtClient = getRTClient();
let unsubscribeTrade: (() => void) | null = null;

// Calcul de la variation de prix
const priceChange = computed(() => {
  const closingPrice = closingPrices.value[selectedTimeframe.value] || 0;
  if (closingPrice === 0) return 0;
  return currentPrice.value - closingPrice;
});

const priceChangePercent = computed(() => {
  const closingPrice = closingPrices.value[selectedTimeframe.value] || 0;
  if (closingPrice === 0) return 0;
  return (priceChange.value / closingPrice) * 100;
});

const isPositive = computed(() => priceChange.value > 0);

// Mise à jour du convertisseur
function updateEurFromBtc() {
  if (currentPrice.value > 0) {
    eurAmount.value = btcAmount.value * currentPrice.value;
  }
}

function updateBtcFromEur() {
  if (currentPrice.value > 0) {
    btcAmount.value = eurAmount.value / currentPrice.value;
  }
}

// Charger les closing prices depuis l'API
async function loadClosingPrices() {
  try {
    console.log(`Loading closing prices for ${realSymbol.value}`);

    // 24h: closing price d'il y a 24h (1h interval, première bougie)
    const data24h = await fetchCandles(realSymbol.value, "1h", 24);
    if (data24h.length > 0) {
      closingPrices.value["24h"] = data24h[0].close;
      console.log(`24h closing: ${data24h[0].close}`);
    }

    // 7d: closing price d'il y a 7 jours (1h interval, 168 heures)
    const data7d = await fetchCandles(realSymbol.value, "1h", 168);
    if (data7d.length > 0) {
      closingPrices.value["7d"] = data7d[0].close;
      console.log(`7d closing: ${data7d[0].close}`);
    }

    // 1M: closing price d'il y a 1 mois (1d interval, 30 jours)
    const data1M = await fetchCandles(realSymbol.value, "1d", 30);
    if (data1M.length > 0) {
      closingPrices.value["1M"] = data1M[0].close;
      console.log(`1M closing: ${data1M[0].close}`);
    }

    // 1Y: closing price d'il y a 1 an (1d interval, 365 jours)
    const data1Y = await fetchCandles(realSymbol.value, "1d", 365);
    if (data1Y.length > 0) {
      closingPrices.value["1Y"] = data1Y[0].close;
      console.log(`1Y closing: ${data1Y[0].close}`);
    }
  } catch (error) {
    console.error("Error loading closing prices:", error);
  }
}

function toggleTimeframeDropdown() {
  isTimeframeDropdownOpen.value = !isTimeframeDropdownOpen.value;
}

function selectTimeframe(timeframe: "24h" | "7d" | "1M" | "1Y") {
  selectedTimeframe.value = timeframe;
  isTimeframeDropdownOpen.value = false;
}

// Lifecycle
onMounted(async () => {
  try {
    // Charger les closing prices
    await loadClosingPrices();

    // Connecter au WebSocket
    if (!rtClient.isConnected()) {
      await rtClient.connect();
    }

    // S'abonner aux trades du symbole
    console.log("📡 Subscribing to trades for:", realSymbol.value);
    rtClient.subscribe("trade", realSymbol.value);

    // Écouter les trades
    unsubscribeTrade = rtClient.on("trade", (message: any) => {
      if (message.data && message.data.symbol === realSymbol.value) {
        const price = message.data.price;

        // Mettre à jour le prix
        previousPrice.value = currentPrice.value;
        currentPrice.value = price;

        // Mettre à jour les hauts/bas SEULEMENT au premier prix
        if (high24h.value === 0 && low24h.value === 0) {
          high24h.value = price;
          low24h.value = price;
        } else {
          // Sinon, update seulement si VRAIMENT dépassé de beaucoup
          // (par exemple > 5% au-dessus/au-dessous)
          if (price > high24h.value * 1.05) {
            high24h.value = price;
          }
          if (price < low24h.value * 0.95) {
            low24h.value = price;
          }
        }

        // Mettre à jour le convertisseur
        eurAmount.value = btcAmount.value * currentPrice.value;

        console.log(
          `📈 ${realSymbol.value}: ${price}€ (${priceChangePercent.value.toFixed(2)}% vs ${selectedTimeframe.value})`
        );
      }
    });
  } catch (error) {
    console.warn("WebSocket connection failed for price panel:", error);
  }
});

onUnmounted(() => {
  if (unsubscribeTrade) {
    unsubscribeTrade();
  }
  if (realSymbol.value) {
    rtClient.unsubscribe("trade", realSymbol.value);
  }
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
          <span class="crypto-price-panel-symbol">{{ realSymbol }}</span>
          <span class="crypto-price-panel-rank">#1</span>
        </div>
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
        <div class="crypto-price-panel-change-content">
          <div class="crypto-price-panel-change-text-wrapper">
            <TrendingUp
              v-if="isPositive"
              class="crypto-price-panel-change-icon"
            />
            <TrendingDown v-else class="crypto-price-panel-change-icon" />
            <span class="change-text">
              {{ isPositive ? "+" : "" }}{{ priceChangePercent.toFixed(2) }}%
            </span>
          </div>
          <!-- Dropdown pour timeframe de variation -->
          <div class="crypto-price-panel-variation-dropdown">
            <button
              class="crypto-price-panel-variation-button"
              @click="toggleTimeframeDropdown"
              :title="`Variation sur ${selectedTimeframe}`"
            >
              <span>{{ selectedTimeframe }}</span>
              <ChevronDown
                class="crypto-price-panel-variation-icon"
                :class="{
                  'crypto-price-panel-variation-icon--open':
                    isTimeframeDropdownOpen,
                }"
                :size="14"
              />
            </button>
            <div
              v-if="isTimeframeDropdownOpen"
              class="crypto-price-panel-variation-menu"
            >
              <button
                v-for="timeframe in timeframeOptions"
                :key="timeframe"
                class="crypto-price-panel-variation-option"
                :class="{
                  'crypto-price-panel-variation-option--active':
                    timeframe === selectedTimeframe,
                }"
                @click="selectTimeframe(timeframe)"
              >
                {{ timeframe }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Convertisseur BTC/EUR -->
    <div class="crypto-price-panel-converter-section">
      <h3 class="crypto-price-panel-section-title">
        Convertisseur {{ realSymbol.split("/")[0] }} / EUR
      </h3>
      <div class="crypto-price-panel-converter">
        <div class="crypto-price-panel-currency-input">
          <label class="crypto-price-panel-currency-label">{{
            realSymbol.split("/")[0]
          }}</label>
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
