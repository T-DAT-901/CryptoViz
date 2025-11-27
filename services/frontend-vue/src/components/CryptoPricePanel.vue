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
const previousPrice = ref(0);

// Hauts/bas par timeframe (pour la section Performance)
const highLowByTimeframe = ref<Record<string, { high: number; low: number }>>({
  "24h": { high: 0, low: 0 },
  "7d": { high: 0, low: 0 },
  "1M": { high: 0, low: 0 },
  "1Y": { high: 0, low: 0 },
});

// Historique des hauts/bas (sur toute la période disponible)
const highHistorical = ref(0);
const lowHistorical = ref(0);
const highHistoricalDate = ref("");
const lowHistoricalDate = ref("");

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

// Extraire le nom de la crypto du symbole (ex: BTC/USDT -> BTC)
const cryptoSymbol = computed(() => {
  return realSymbol.value.split("/")[0];
});

// Utiliser le cryptoName passé en prop ou extraire du symbole
const displayName = computed(() => {
  if (props.cryptoName) {
    return props.cryptoName;
  }
  // Fallback: afficher le symbole
  return cryptoSymbol.value;
});

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

// Hauts/bas du timeframe sélectionné
const currentHighLow = computed(() => {
  return (
    highLowByTimeframe.value[selectedTimeframe.value] || { high: 0, low: 0 }
  );
});

// Mise à jour du convertisseur
function updateEurFromBtc() {
  if (currentPrice.value > 0 && btcAmount.value >= 0) {
    eurAmount.value = btcAmount.value * currentPrice.value;
  }
}

function updateBtcFromEur() {
  if (currentPrice.value > 0 && eurAmount.value >= 0) {
    btcAmount.value = eurAmount.value / currentPrice.value;
  }
}

// Charger les prix de référence pour chaque timeframe
async function loadClosingPrices() {
  try {
    console.log(`Loading reference prices for ${realSymbol.value}`);

    // 24h: prix d'il y a 24h (dernière bougie 1h qui couvre il y a 24h)
    const data24h = await fetchCandles(realSymbol.value, "1h", 24);
    if (data24h.length > 0) {
      closingPrices.value["24h"] = data24h[0].close;
      // Charger aussi les hauts/bas pour 24h
      let high24h = 0;
      let low24h = Number.MAX_VALUE;
      data24h.forEach((candle: any) => {
        high24h = Math.max(high24h, candle.high);
        low24h = Math.min(low24h, candle.low);
      });
      highLowByTimeframe.value["24h"] = { high: high24h, low: low24h };
      console.log(
        `24h reference price: ${data24h[0].close}, high: ${high24h}, low: ${low24h}`
      );
    }

    // 7d: prix d'il y a 7 jours (dernière bougie 1h qui couvre il y a 7 jours)
    const data7d = await fetchCandles(realSymbol.value, "1h", 168);
    if (data7d.length > 0) {
      closingPrices.value["7d"] = data7d[0].close;
      // Charger aussi les hauts/bas pour 7d
      let high7d = 0;
      let low7d = Number.MAX_VALUE;
      data7d.forEach((candle: any) => {
        high7d = Math.max(high7d, candle.high);
        low7d = Math.min(low7d, candle.low);
      });
      highLowByTimeframe.value["7d"] = { high: high7d, low: low7d };
      console.log(
        `7d reference price: ${data7d[0].close}, high: ${high7d}, low: ${low7d}`
      );
    }

    // 1M: prix d'il y a 1 mois (première bougie des 60 jours)
    const data1M = await fetchCandles(realSymbol.value, "1d", 60);
    if (data1M.length > 0) {
      // Prendre le premier élément (le plus ancien) comme prix de référence
      closingPrices.value["1M"] = data1M[0].close;
      // Charger aussi les hauts/bas pour 1M
      let high1M = 0;
      let low1M = Number.MAX_VALUE;
      data1M.forEach((candle: any) => {
        high1M = Math.max(high1M, candle.high);
        low1M = Math.min(low1M, candle.low);
      });
      highLowByTimeframe.value["1M"] = { high: high1M, low: low1M };
      console.log(
        `1M reference price: ${data1M[0].close}, high: ${high1M}, low: ${low1M}`
      );
    }

    // 1Y: prix d'il y a 1 an (première bougie des 730 jours)
    const data1Y = await fetchCandles(realSymbol.value, "1d", 730);
    if (data1Y.length > 0) {
      // Prendre le premier élément (le plus ancien) comme prix de référence
      closingPrices.value["1Y"] = data1Y[0].close;
      // Charger aussi les hauts/bas pour 1Y
      let high1Y = 0;
      let low1Y = Number.MAX_VALUE;
      data1Y.forEach((candle: any) => {
        high1Y = Math.max(high1Y, candle.high);
        low1Y = Math.min(low1Y, candle.low);
      });
      highLowByTimeframe.value["1Y"] = { high: high1Y, low: low1Y };
      console.log(
        `1Y reference price: ${data1Y[0].close}, high: ${high1Y}, low: ${low1Y}`
      );
    }
  } catch (error) {
    console.error("Error loading reference prices:", error);
  }
}

// Charger les hauts/bas historiques
async function loadHistoricalPrices() {
  try {
    console.log(`Loading historical prices for ${realSymbol.value}`);

    // Récupérer les bougies sur 2 ans pour trouver le haut et bas historiques complets
    const dataHistorical = await fetchCandles(realSymbol.value, "1d", 730);

    if (dataHistorical.length > 0) {
      // Trouver le haut et bas sur tous les historiques disponibles
      let maxHigh = 0;
      let minLow = Number.MAX_VALUE;
      let maxHighDate = "";
      let minLowDate = "";

      dataHistorical.forEach((candle: any) => {
        if (candle.high > maxHigh) {
          maxHigh = candle.high;
          maxHighDate = new Date(candle.time).toLocaleDateString("fr-FR", {
            month: "short",
            day: "numeric",
            year: "numeric",
          });
        }
        if (candle.low < minLow) {
          minLow = candle.low;
          minLowDate = new Date(candle.time).toLocaleDateString("fr-FR", {
            month: "short",
            day: "numeric",
            year: "numeric",
          });
        }
      });

      highHistorical.value = maxHigh;
      lowHistorical.value = minLow;
      highHistoricalDate.value = maxHighDate;
      lowHistoricalDate.value = minLowDate;

      console.log(`Historical high: €${maxHigh} on ${maxHighDate}`);
      console.log(`Historical low: €${minLow} on ${minLowDate}`);
    }
  } catch (error) {
    console.error("Error loading historical prices:", error);
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
    // Charger les closing prices et historiques
    await loadClosingPrices();
    await loadHistoricalPrices();

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

        // Mettre à jour le convertisseur
        eurAmount.value = btcAmount.value * currentPrice.value;

        // console.log(
        //   `📈 ${realSymbol.value}: ${price}€ (${priceChangePercent.value.toFixed(2)}% vs ${selectedTimeframe.value})`
        // );
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

// Watcher pour mettre à jour le convertisseur EUR quand le prix change
watch(currentPrice, () => {
  if (btcAmount.value > 0 && currentPrice.value > 0) {
    eurAmount.value = btcAmount.value * currentPrice.value;
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
          <h1 class="crypto-price-panel-name">{{ displayName }}</h1>
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
        Convertisseur {{ cryptoSymbol }} / EUR
      </h3>
      <div class="crypto-price-panel-converter">
        <div class="crypto-price-panel-currency-input">
          <label class="crypto-price-panel-currency-label">{{
            cryptoSymbol
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
        <select
          v-model="selectedTimeframe"
          class="crypto-price-panel-timeframe-select"
        >
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
              currentHighLow.low.toLocaleString("fr-FR", {
                minimumFractionDigits: 2,
              })
            }}</span
          >
        </div>
        <div class="crypto-price-panel-stat-row">
          <span class="crypto-price-panel-stat-label">Haut</span>
          <span class="crypto-price-panel-stat-value"
            >€{{
              currentHighLow.high.toLocaleString("fr-FR", {
                minimumFractionDigits: 2,
              })
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
              width:
                ((currentPrice - currentHighLow.low) /
                  (currentHighLow.high - currentHighLow.low)) *
                  100 +
                '%',
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
          <span class="crypto-price-panel-historical-price">
            €{{
              highHistorical.toLocaleString("fr-FR", {
                minimumFractionDigits: 2,
                maximumFractionDigits: 2,
              })
            }}</span
          >
        </div>
        <div class="crypto-price-panel-historical-meta">
          <span class="crypto-price-panel-historical-date">{{
            highHistoricalDate
          }}</span>
          <span
            class="crypto-price-panel-historical-change"
            :class="{
              'crypto-price-panel-historical-change--negative':
                currentPrice < highHistorical,
              'crypto-price-panel-historical-change--positive':
                currentPrice >= highHistorical,
            }"
          >
            {{
              currentPrice > 0
                ? (
                    ((currentPrice - highHistorical) / highHistorical) *
                    100
                  ).toFixed(2)
                : "0"
            }}%
          </span>
        </div>
      </div>

      <div class="crypto-price-panel-historical-item">
        <div class="crypto-price-panel-historical-main">
          <span class="crypto-price-panel-historical-label"
            >Bas historique</span
          >
          <span class="crypto-price-panel-historical-price">
            €{{
              lowHistorical.toLocaleString("fr-FR", {
                minimumFractionDigits: 2,
                maximumFractionDigits: 8,
              })
            }}</span
          >
        </div>
        <div class="crypto-price-panel-historical-meta">
          <span class="crypto-price-panel-historical-date">{{
            lowHistoricalDate
          }}</span>
          <span
            class="crypto-price-panel-historical-change crypto-price-panel-historical-change--positive"
          >
            {{
              currentPrice > 0
                ? (
                    ((currentPrice - lowHistorical) / lowHistorical) *
                    100
                  ).toFixed(2)
                : "0"
            }}%
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
