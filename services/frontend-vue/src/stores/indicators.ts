import { defineStore } from "pinia";

export const useIndicatorsStore = defineStore("indicators", {
  state: () => ({
    // Layout mode: 'compact' (unified chart) or 'detailed' (separate charts)
    layoutMode: "detailed" as "compact" | "detailed",

    // Timeframe synchronization - new granular timeframes
    selectedTimeframe: "1h" as "1m" | "5m" | "15m" | "1h" | "24h",

    // visibility
    showRSI: true,
    showMACD: true,
    showBollinger: true,
    showMomentum: true,

    // default parameters
    rsiPeriod: 14,
    macdFast: 12,
    macdSlow: 26,
    macdSignal: 9,
    bbPeriod: 20,
    bbStd: 2,
    momPeriod: 10,
  }),

  actions: {
    toggleLayoutMode() {
      this.layoutMode = this.layoutMode === "compact" ? "detailed" : "compact";
    },

    setLayoutMode(mode: "compact" | "detailed") {
      this.layoutMode = mode;
    },

    setTimeframe(timeframe: "1m" | "5m" | "15m" | "1h" | "24h") {
      this.selectedTimeframe = timeframe;
    },
  },
});
