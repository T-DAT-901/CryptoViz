<script setup lang="ts">
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
  <section class="panel">
    <header class="head">News</header>
    <ul class="list">
      <li v-for="n in items" :key="n.id" class="item">
        <div class="title">{{ n.title }}</div>
        <div class="meta">
          <span class="src">{{ n.source || "source" }}</span>
          <span class="dot">•</span>
          <span class="t">{{ fmt(n.time) }}</span>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.panel {
  background: #0b0e11;
  border: 1px solid #1a1f24;
  border-radius: 10px;
  padding: 12px;
}
.head {
  font-weight: 700;
  color: #e5e7eb;
  margin-bottom: 8px;
}
.list {
  display: grid;
  gap: 10px;
}
.item {
  padding: 10px;
  border: 1px solid #141a1f;
  border-radius: 8px;
  background: #0c1116;
}
.title {
  color: #e5e7eb;
  line-height: 1.3;
}
.meta {
  color: #9ca3af;
  font-size: 12px;
  margin-top: 6px;
  display: flex;
  gap: 8px;
  align-items: center;
}
.dot {
  opacity: 0.5;
}
</style>
