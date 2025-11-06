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
    <div class="sentiment-header">
      <div class="header-left">
        <!-- Icône de chart Lucide -->
        <BarChart3 class="chart-icon" />
        <div class="title-section">
          <h3 class="title">Community sentiment</h3>
          <span class="vote-count">{{ formattedVotes }} votes</span>
        </div>
      </div>

      <div class="pagination">
        <button
          class="nav-btn"
          @click="navigatePage('prev')"
          :disabled="sentimentData.currentPage === 1"
        >
          <ChevronLeft />
        </button>
        <span class="page-info"
          >{{ sentimentData.currentPage }}/{{ sentimentData.totalPages }}</span
        >
        <button
          class="nav-btn"
          @click="navigatePage('next')"
          :disabled="sentimentData.currentPage === sentimentData.totalPages"
        >
          <ChevronRight />
        </button>
      </div>
    </div>

    <!-- Barre de sentiment avec pourcentages -->
    <div class="sentiment-bar-container">
      <div class="percentage-row">
        <div class="bullish-section">
          <!-- Icône trending up Lucide -->
          <TrendingUp class="trend-icon bullish" />
          <span class="percentage">{{ bullishPercentage }}%</span>
        </div>
        <div class="bearish-section">
          <span class="percentage">{{ bearishPercentage }}%</span>
          <!-- Icône trending down Lucide -->
          <TrendingDown class="trend-icon bearish" />
        </div>
      </div>

      <!-- Barre de progression raffinée -->
      <div class="sentiment-bar">
        <div
          class="bullish-fill"
          :style="{ width: bullishPercentage + '%' }"
        ></div>
        <div
          class="bearish-fill"
          :style="{ width: bearishPercentage + '%' }"
        ></div>
      </div>
    </div>

    <!-- Boutons de vote raffinés -->
    <div class="vote-buttons">
      <button class="vote-btn bullish">
        <TrendingUp class="vote-icon" />
        <span class="vote-label">Bullish</span>
      </button>
      <button class="vote-btn bearish">
        <TrendingDown class="vote-icon" />
        <span class="vote-label">Bearish</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.community-sentiment {
  background: #040d12;
  border-radius: 12px;
  padding: 18px;
  color: #e5e7eb;
  border: 1px solid #1a1a2e;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

/* Header */
.sentiment-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.chart-icon {
  width: 16px;
  height: 16px;
  color: #9ca3af;
  stroke: currentColor;
  stroke-width: 2;
  fill: none;
}

.title-section {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.title {
  font-size: 15px;
  font-weight: 500;
  margin: 0;
  color: #e5e7eb;
  letter-spacing: -0.01em;
}

.vote-count {
  font-size: 11px;
  color: #6b7280;
  font-weight: 400;
}

.pagination {
  display: flex;
  align-items: center;
  gap: 6px;
}

.nav-btn {
  background: transparent;
  border: 1px solid #374151;
  border-radius: 4px;
  color: #9ca3af;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
}

.nav-btn svg {
  width: 12px;
  height: 12px;
}

.nav-btn:hover:not(:disabled) {
  background: rgba(55, 65, 81, 0.5);
  border-color: #6b7280;
  color: #e5e7eb;
}

.nav-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-info {
  font-size: 11px;
  color: #9ca3af;
  min-width: 26px;
  text-align: center;
  font-weight: 500;
}

/* Barre de sentiment */
.sentiment-bar-container {
  margin-bottom: 18px;
}

.percentage-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.bullish-section,
.bearish-section {
  display: flex;
  align-items: center;
  gap: 6px;
}

.trend-icon {
  width: 14px;
  height: 14px;
  stroke: currentColor;
  stroke-width: 2;
  fill: none;
}

.trend-icon.bullish {
  color: #10b981;
}

.trend-icon.bearish {
  color: #ef4444;
}

.percentage {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.bullish-section .percentage {
  color: #10b981;
}

.bearish-section .percentage {
  color: #ef4444;
}

.sentiment-bar {
  height: 6px;
  border-radius: 3px;
  overflow: hidden;
  display: flex;
  background: #1a1a2e;
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.1);
}

.bullish-fill {
  background: linear-gradient(90deg, #10b981, #059669);
  transition: width 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.bearish-fill {
  background: linear-gradient(90deg, #ef4444, #dc2626);
  transition: width 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

/* Boutons de vote */
.vote-buttons {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.vote-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 14px;
  border-radius: 8px;
  border: 1.5px solid;
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 13px;
  font-weight: 500;
  position: relative;
  overflow: hidden;
}

.vote-btn::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.vote-btn.bullish {
  border-color: #10b981;
  color: #10b981;
}

.vote-btn.bullish::before {
  background: linear-gradient(
    135deg,
    rgba(16, 185, 129, 0.05),
    rgba(16, 185, 129, 0.1)
  );
}

.vote-btn.bullish:hover::before {
  opacity: 1;
}

.vote-btn.bearish {
  border-color: #ef4444;
  color: #ef4444;
}

.vote-btn.bearish::before {
  background: linear-gradient(
    135deg,
    rgba(239, 68, 68, 0.05),
    rgba(239, 68, 68, 0.1)
  );
}

.vote-btn.bearish:hover::before {
  opacity: 1;
}

.vote-icon {
  width: 14px;
  height: 14px;
  stroke: currentColor;
  stroke-width: 2;
  fill: none;
}

.vote-label {
  font-weight: 500;
  letter-spacing: -0.01em;
}

/* Responsive */
@media (max-width: 768px) {
  .community-sentiment {
    padding: 14px;
  }

  .sentiment-header {
    margin-bottom: 20px;
  }

  .title {
    font-size: 14px;
  }

  .vote-count {
    font-size: 10px;
  }

  .percentage {
    font-size: 16px;
  }

  .vote-btn {
    padding: 8px 12px;
    font-size: 12px;
  }

  .vote-icon {
    width: 12px;
    height: 12px;
  }
}
</style>
