<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { BarChart3 } from "lucide-vue-next";
import { NewsService, type NewsArticle } from "@/services/news.api";

const props = defineProps<{
  symbol?: string;
}>();

const totalArticles = ref(0);
const averageSentiment = ref(0); // Score de -1 (bearish) à 1 (bullish)
const isLoading = ref(true);

async function fetchSentimentData(symbol: string) {
  if (!symbol) return;

  isLoading.value = true;
  // Extrait le token de base (ex: BTC de BTC/USDT) et le met en minuscule
  const baseToken = symbol.split('/')[0].toLowerCase();
  console.log(`[Sentiment] 1. Fetching sentiment for base token: ${baseToken}`);

  try {
    const fetchedArticles = await NewsService.getNewsBySymbol(baseToken, 500);
    console.log(`[Sentiment] 2. Fetched ${fetchedArticles.length} articles from the API for token '${baseToken}'.`);

    const twentyFourHoursAgo = new Date();
    twentyFourHoursAgo.setHours(twentyFourHoursAgo.getHours() - 24);

    const recentArticles = fetchedArticles.filter(
      (article) =>
        new Date(article.time) > twentyFourHoursAgo &&
        article.sentiment_score !== null
    );
    console.log(`[Sentiment] 3. Found ${recentArticles.length} articles from the last 24 hours with a sentiment score.`);

    totalArticles.value = recentArticles.length;

    if (recentArticles.length > 0) {
      const sentimentSum = recentArticles.reduce(
        (sum, article) => sum + article.sentiment_score!,
        0
      );
      averageSentiment.value = sentimentSum / recentArticles.length;
      console.log(`[Sentiment] 4. Calculated average sentiment: ${averageSentiment.value}`);
    } else {
      averageSentiment.value = 0; // Neutre si pas d'articles
      console.log(`[Sentiment] 4. No recent articles with score found, sentiment is neutral (0).`);
    }
  } catch (error) {
    console.error(`[Sentiment] Failed to fetch sentiment data for ${baseToken}:`, error);
    totalArticles.value = 0;
    averageSentiment.value = 0;
  } finally {
    isLoading.value = false;
  }
}

// Convertit le score de [-1, 1] à un pourcentage de [0, 100] pour la barre
const sentimentBarPercentage = computed(() => {
  return (averageSentiment.value + 1) / 2 * 100;
});

const sentimentLabel = computed(() => {
  if (averageSentiment.value > 0.1) return "Plutôt positif";
  if (averageSentiment.value < -0.1) return "Plutôt négatif";
  return "Neutre";
});

const sentimentColor = computed(() => {
  if (averageSentiment.value > 0.1) return "var(--accent-green)";
  if (averageSentiment.value < -0.1) return "var(--accent-red)";
  return "var(--text-secondary)";
});

onMounted(() => {
  if (props.symbol) {
    fetchSentimentData(props.symbol);
  }
});

watch(
  () => props.symbol,
  (newSymbol) => {
    if (newSymbol) {
      fetchSentimentData(newSymbol);
    }
  }
);
</script>

<template>
  <div class="community-sentiment">
    <div class="community-sentiment-header">
      <div class="community-sentiment-header-left">
        <BarChart3 class="community-sentiment-icon" />
        <div class="community-sentiment-title-section">
          <h3 class="community-sentiment-title">Analyse de Sentiment (24h)</h3>
          <span v-if="!isLoading" class="community-sentiment-vote-count">
            {{ totalArticles }} articles analysés
          </span>
          <span v-else class="community-sentiment-vote-count">Loading...</span>
        </div>
      </div>
    </div>

    <div v-if="!isLoading" class="sentiment-analysis-section">
       <div class="sentiment-bar-container">
        <div class="sentiment-bar-labels">
          <span>Négatif</span>
          <span>Positif</span>
        </div>
        <div class="sentiment-bar">
          <div class="sentiment-bar-background"></div>
          <div
            class="sentiment-bar-indicator"
            :style="{ left: `${sentimentBarPercentage}%`, backgroundColor: sentimentColor }"
          ></div>
        </div>
      </div>
       <div class="sentiment-summary">
        <span :style="{ color: sentimentColor }">{{ sentimentLabel }}</span>
      </div>
    </div>
    <div v-else class="loading-shimmer" style="height: 60px; margin-top: 10px;"></div>
  </div>
</template>

<style scoped>
.sentiment-analysis-section {
  padding-top: 1rem;
}

.sentiment-bar-container {
  width: 100%;
}

.sentiment-bar-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.75rem;
  color: var(--text-secondary);
  margin-bottom: 0.5rem;
}

.sentiment-bar {
  position: relative;
  width: 100%;
  height: 8px;
}

.sentiment-bar-background {
  width: 100%;
  height: 100%;
  background: linear-gradient(to right, var(--accent-red), var(--accent-orange), var(--accent-green));
  border-radius: 4px;
}

.sentiment-bar-indicator {
  position: absolute;
  top: -4px; /* Center on the bar */
  width: 16px;
  height: 16px;
  border-radius: 50%;
  transform: translateX(-50%);
  border: 2px solid var(--bg-primary);
  box-shadow: 0 0 5px rgba(0, 0, 0, 0.2);
}

.sentiment-summary {
  text-align: center;
  margin-top: 1rem;
  font-weight: 500;
}
</style>
