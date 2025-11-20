<script setup lang="ts">
import { Newspaper, Clock } from "lucide-vue-next";

type News = {
  id: string;
  time: number;
  title: string;
  source?: string;
  url?: string;
};
const items: News[] = [
  {
    id: "n1",
    time: Date.now() - 3 * 60_000,
    title: "BTC holds gains as liquidity improves",
    source: "CoinDesk",
  },
  {
    id: "n2",
    time: Date.now() - 12 * 60_000,
    title: "ETF inflows drive late-session rally",
    source: "Bloomberg",
  },
  {
    id: "n3",
    time: Date.now() - 25 * 60_000,
    title: "Derivatives data hints at squeeze risk",
    source: "The Block",
  },
];
function fmt(ts: number) {
  return new Date(ts).toLocaleTimeString();
}
</script>

<template>
  <section class="news-feed-panel">
    <header class="news-feed-header">
      <Newspaper class="news-feed-icon" />
      News
    </header>
    <ul class="news-feed-list">
      <li v-for="n in items" :key="n.id" class="news-feed-item">
        <div class="news-feed-title">{{ n.title }}</div>
        <div class="news-feed-meta">
          <span class="news-feed-source">{{ n.source || "source" }}</span>
          <span class="news-feed-dot">•</span>
          <Clock class="news-feed-time-icon" />
          <span class="news-feed-time">{{ fmt(n.time) }}</span>
        </div>
      </li>
    </ul>
  </section>
</template>
