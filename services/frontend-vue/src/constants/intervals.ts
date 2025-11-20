/**
 * Supported aggregation intervals for CryptoViz
 * These intervals match the backend configuration and Kafka topics
 */

export interface IntervalConfig {
  value: string;
  label: string;
  description: string;
  milliseconds: number;
}

/**
 * All supported intervals across the CryptoViz stack
 * Intervals: 1m, 5m, 15m, 1h, 1d
 */
export const SUPPORTED_INTERVALS: IntervalConfig[] = [
  {
    value: "1m",
    label: "1m",
    description: "1 Minute",
    milliseconds: 60 * 1000,
  },
  {
    value: "5m",
    label: "5m",
    description: "5 Minutes",
    milliseconds: 5 * 60 * 1000,
  },
  {
    value: "15m",
    label: "15m",
    description: "15 Minutes",
    milliseconds: 15 * 60 * 1000,
  },
  {
    value: "1h",
    label: "1h",
    description: "1 Hour",
    milliseconds: 60 * 60 * 1000,
  },
  {
    value: "1d",
    label: "1d",
    description: "1 Day",
    milliseconds: 24 * 60 * 60 * 1000,
  },
];

/**
 * Default interval for the application
 */
export const DEFAULT_INTERVAL = "1m";

/**
 * Get interval configuration by value
 */
export function getIntervalConfig(value: string): IntervalConfig | undefined {
  return SUPPORTED_INTERVALS.find((interval) => interval.value === value);
}

/**
 * Check if an interval is supported
 */
export function isValidInterval(value: string): boolean {
  return SUPPORTED_INTERVALS.some((interval) => interval.value === value);
}

/**
 * Get all interval values as an array
 */
export function getIntervalValues(): string[] {
  return SUPPORTED_INTERVALS.map((interval) => interval.value);
}
