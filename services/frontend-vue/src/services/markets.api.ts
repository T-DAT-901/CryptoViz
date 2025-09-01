// récupération des données de backend
export type TickerDTO = {
  symbol: string;
  price: number;
  ts: number;
};
export type CandleDTO = {
  t: number;
  o: number;
  h: number;
  l: number;
  c: number;
};
export type NewsDTO = {
  id: string;
  title: string;
  rel: string;
  publishAt: string;
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
