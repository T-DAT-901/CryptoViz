import { defineStore } from "pinia";

export const useIndicatorsStore = defineStore("indicators", {
  state: () => ({
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
});
