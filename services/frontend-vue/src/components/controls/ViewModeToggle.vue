<script setup lang="ts">
import { LayoutGrid, BarChart3 } from "lucide-vue-next";
import { useIndicatorsStore } from "@/stores/indicators";

const indicatorsStore = useIndicatorsStore();

const toggleMode = () => {
  indicatorsStore.toggleLayoutMode();
};
</script>

<template>
  <div class="view-mode-toggle">
    <button
      class="view-mode-toggle__button"
      :class="{
        'view-mode-toggle__button--active':
          indicatorsStore.layoutMode === 'compact',
      }"
      @click="indicatorsStore.setLayoutMode('compact')"
      title="Mode Compact - Graphique unifié"
    >
      <LayoutGrid class="view-mode-toggle__icon" />
      <span class="view-mode-toggle__label">Compact</span>
    </button>

    <button
      class="view-mode-toggle__button"
      :class="{
        'view-mode-toggle__button--active':
          indicatorsStore.layoutMode === 'detailed',
      }"
      @click="indicatorsStore.setLayoutMode('detailed')"
      title="Mode Détaillé - Graphiques séparés"
    >
      <BarChart3 class="view-mode-toggle__icon" />
      <span class="view-mode-toggle__label">Détaillé</span>
    </button>
  </div>
</template>
