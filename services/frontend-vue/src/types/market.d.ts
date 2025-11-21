export type TickerDTO = {
  symbol: string;
  price: number;
  ts: number;
};
export type CandleDTO = {
  time: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
};
export type NewsDTO = {
  title: string;
  link: string;
  published: string;
  source: string;
  image?: string;
};
export type RSIPoint = {
  ts: number;
  value: number;
};
export type MACDPoint = {
  ts: number;
  macd: number;
  signal: number;
  hist: number;
};
export type BollingerPoint = {
  ts: number;
  upper: number;
  middle: number;
  lower: number;
};
export type MomentumPoint = {
  ts: number;
  value: number;
};
