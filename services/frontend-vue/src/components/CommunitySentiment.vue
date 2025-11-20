<script setup lang="ts">
import { ref, computed } from "vue";
import {
  BarChart3,
  TrendingUp,
  TrendingDown,
  ChevronLeft,
  ChevronRight,
} from "lucide-vue-next";

const props = defineProps<{
  symbol?: string;
}>();

// Données de sentiment (mockées pour l'exemple)
const sentimentData = ref({
  totalVotes: 5200000, // 5.2M votes
  bullishVotes: 4264000, // 82%
  bearishVotes: 936000, // 18%
  currentPage: 1,
  totalPages: 2,
});

// Calculs des pourcentages
const bullishPercentage = computed(() => {
  const total =
    sentimentData.value.bullishVotes + sentimentData.value.bearishVotes;
  return Math.round((sentimentData.value.bullishVotes / total) * 100);
});

const bearishPercentage = computed(() => {
  const total =
    sentimentData.value.bullishVotes + sentimentData.value.bearishVotes;
  return Math.round((sentimentData.value.bearishVotes / total) * 100);
});

// Formatage du nombre de votes
const formattedVotes = computed(() => {
  const votes = sentimentData.value.totalVotes;
  if (votes >= 1000000) {
    return (votes / 1000000).toFixed(1) + "M";
  } else if (votes >= 1000) {
    return (votes / 1000).toFixed(1) + "K";
  }
  return votes.toString();
});

// Navigation entre pages
function navigatePage(direction: "prev" | "next") {
  if (direction === "prev" && sentimentData.value.currentPage > 1) {
    sentimentData.value.currentPage--;
  } else if (
    direction === "next" &&
    sentimentData.value.currentPage < sentimentData.value.totalPages
  ) {
    sentimentData.value.currentPage++;
  }
}
</script>

<template>
  <div class="community-sentiment">
    <!-- Header avec navigation -->
    <div class="community-sentiment-header">
      <div class="community-sentiment-header-left">
        <!-- Icône de chart Lucide -->
        <BarChart3 class="community-sentiment-icon" />
        <div class="community-sentiment-title-section">
          <h3 class="community-sentiment-title">Sentiment</h3>
          <span class="community-sentiment-vote-count"
            >{{ formattedVotes }} votes</span
          >
        </div>
      </div>

      <div class="community-sentiment-pagination">
        <button
          class="community-sentiment-nav-btn"
          @click="navigatePage('prev')"
          :disabled="sentimentData.currentPage === 1"
        >
          <ChevronLeft />
        </button>
        <span class="community-sentiment-page-info"
          >{{ sentimentData.currentPage }}/{{ sentimentData.totalPages }}</span
        >
        <button
          class="community-sentiment-nav-btn"
          @click="navigatePage('next')"
          :disabled="sentimentData.currentPage === sentimentData.totalPages"
        >
          <ChevronRight />
        </button>
      </div>
    </div>

    <!-- Barre de sentiment avec pourcentages -->
    <div class="community-sentiment-sentiment-section">
      <div class="community-sentiment-percentage-row">
        <div class="community-sentiment-bullish-section">
          <!-- Icône trending up Lucide -->
          <TrendingUp
            class="community-sentiment-trend-icon community-sentiment-trend-icon--bullish"
          />
          <span class="community-sentiment-percentage"
            >{{ bullishPercentage }}%</span
          >
        </div>
        <div class="community-sentiment-bearish-section">
          <span class="community-sentiment-percentage"
            >{{ bearishPercentage }}%</span
          >
          <!-- Icône trending down Lucide -->
          <TrendingDown
            class="community-sentiment-trend-icon community-sentiment-trend-icon--bearish"
          />
        </div>
      </div>

      <!-- Barre de progression raffinée -->
      <div class="community-sentiment-bar">
        <div
          class="community-sentiment-bullish-fill"
          :style="{ width: bullishPercentage + '%' }"
        ></div>
        <div
          class="community-sentiment-bearish-fill"
          :style="{ width: bearishPercentage + '%' }"
        ></div>
      </div>
    </div>

    <!-- Boutons de vote raffinés -->
    <div class="community-sentiment-vote-buttons">
      <button
        class="community-sentiment-vote-btn community-sentiment-vote-btn--bullish"
      >
        <TrendingUp class="community-sentiment-vote-icon" />
        <span class="community-sentiment-vote-label">Bullish</span>
      </button>
      <button
        class="community-sentiment-vote-btn community-sentiment-vote-btn--bearish"
      >
        <TrendingDown class="community-sentiment-vote-icon" />
        <span class="community-sentiment-vote-label">Bearish</span>
      </button>
    </div>
  </div>
</template>
