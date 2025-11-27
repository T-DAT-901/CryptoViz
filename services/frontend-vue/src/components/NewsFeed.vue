<script setup lang="ts">
import { ref, onMounted } from "vue";
import { Newspaper, Clock, ExternalLink } from "lucide-vue-next";
import { fetchNews } from "@/services/markets.api";
import type { NewsDTO } from "@/types/market";

const news = ref<NewsDTO[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

onMounted(async () => {
  loading.value = true;
  try {
    news.value = await fetchNews();
  } catch (err) {
    error.value = "Failed to load news";
    console.error(err);
  } finally {
    loading.value = false;
  }
});

function formatTime(isoString: string): string {
  const date = new Date(isoString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}
</script>

<template>
  <section class="news-feed-panel">
    <header class="news-feed-header">
      <Newspaper class="news-feed-icon" />
      News
    </header>

    <div v-if="loading" class="news-feed-loading">Loading news...</div>

    <div v-else-if="error" class="news-feed-error">
      {{ error }}
    </div>

    <ul v-else class="news-feed-list">
      <li
        v-for="(item, idx) in news"
        :key="`${item.source}-${idx}`"
        class="news-feed-item"
      >
        <div class="news-feed-content">
          <a
            :href="item.url"
            target="_blank"
            rel="noopener noreferrer"
            class="news-feed-title"
          >
            {{ item.title }}
            <ExternalLink class="news-feed-link-icon" />
          </a>
          <div class="news-feed-meta">
            <span class="news-feed-source">{{ item.source }}</span>
            <span class="news-feed-dot">•</span>
            <Clock class="news-feed-time-icon" />
            <span class="news-feed-time">{{ formatTime(item.time) }}</span>
          </div>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.news-feed-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-card);
  border: 1px solid var(--border-primary);
  border-radius: var(--radius-lg);
  padding: 0;
  overflow: hidden;
}

.news-feed-header {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-md) var(--space-lg);
  font-weight: var(--font-weight-semibold);
  font-size: var(--font-size-base);
  border-bottom: 1px solid var(--border-primary);
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.news-feed-icon {
  width: 16px;
  height: 16px;
  color: var(--accent-blue);
}

.news-feed-list {
  list-style: none;
  padding: 0;
  margin: 0;
  overflow-y: auto;
  flex: 1;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: var(--bg-primary);
  }

  &::-webkit-scrollbar-thumb {
    background: var(--bg-secondary);
    border-radius: var(--radius-md);

    &:hover {
      background: var(--bg-interactive);
    }
  }
}

.news-feed-item {
  padding: var(--space-md) var(--space-lg);
  border-bottom: 1px solid var(--border-primary);
  transition: var(--transition-base);

  &:hover {
    background: var(--bg-secondary);
  }

  &:last-child {
    border-bottom: none;
  }
}

.news-feed-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.news-feed-title {
  display: flex;
  align-items: flex-start;
  gap: var(--space-sm);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--text-primary);
  text-decoration: none;
  line-height: var(--line-height-normal);
  transition: var(--transition-base);

  &:hover {
    color: var(--accent-blue);
  }
}

.news-feed-link-icon {
  width: 12px;
  height: 12px;
  color: var(--accent-blue);
  flex-shrink: 0;
  margin-top: 2px;
  opacity: 0.6;
  transition: var(--transition-base);
}

.news-feed-title:hover .news-feed-link-icon {
  opacity: 1;
}

.news-feed-meta {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.news-feed-source {
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
}

.news-feed-dot {
  color: var(--text-muted);
}

.news-feed-time-icon {
  width: 10px;
  height: 10px;
  color: var(--text-muted);
}

.news-feed-time {
  color: var(--text-muted);
}

.news-feed-loading,
.news-feed-error {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-2xl) var(--space-lg);
  text-align: center;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.news-feed-error {
  color: var(--accent-red);
}
</style>
