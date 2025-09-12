<script setup lang="ts">
import { ref, onMounted } from "vue";
import newsMock from "@/services/mocks/news.json";

interface Article {
  title: string;
  link: string;
  published: string; // ISO string
  source: string;
  image: string;
}

const articles = ref<Article[]>([]);

onMounted(() => {
  articles.value = newsMock as Article[];
});

function formatTime(isoDate: string): string {
  const published = new Date(isoDate).getTime();
  const now = Date.now();
  const diff = Math.floor((now - published) / 1000); // in seconds

  if (diff < 60) return "il y a moins d'1 minute";
  if (diff < 3600) return `il y a ${Math.floor(diff / 60)} minutes`;
  if (diff < 86400) return `il y a ${Math.floor(diff / 3600)} heures`;

  const d = new Date(isoDate);
  if (diff < 172800) {
    // less than 2 days → show "11 sept. à 14h30"
    return `${d.getDate()} ${d.toLocaleString("fr-FR", {
      month: "short",
    })}. à ${d.getHours()}h${d.getMinutes().toString().padStart(2, "0")}`;
  }

  return d.toLocaleDateString("fr-FR", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}
</script>

<template>
  <div class="news-feed">
    <ul>
      <li v-for="(a, i) in articles" :key="i" class="article">
        <img :src="a.image" alt="thumb" class="thumb" />
        <div class="content">
          <span class="time">{{ formatTime(a.published) }}</span>
          <a :href="a.link" target="_blank" class="title">{{ a.title }}</a>
          <span class="source">{{ a.source }}</span>
        </div>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.news-feed ul {
  padding: 0;
  margin: 0;
  list-style: none; /* remove bullet points */
}

.article {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.thumb {
  width: 60px;
  height: 60px;
  object-fit: cover;
  border-radius: 6px;
  flex-shrink: 0;
}

.content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.time {
  font-size: 12px;
  color: #9ca3af;
  margin-bottom: 4px;
}

.title {
  font-weight: 600;
  color: #e5e7eb;
  text-decoration: none;
}

.title:hover {
  text-decoration: underline;
}

.source {
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}
</style>
