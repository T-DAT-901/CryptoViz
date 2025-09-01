import os, asyncio, signal, time
import orjson

# uvloop facultatif
try:
    import uvloop  # type: ignore
    uvloop.install()
except Exception:
    pass

import ccxt.pro as ccxtpro  # ccxt.pro requis pour watch_*
from aiokafka import AIOKafkaProducer

EXCHANGE_NAME = "BINANCE"


# ----------------------------- utils ---------------------------------
def dumps(obj: dict) -> bytes:
    return orjson.dumps(obj)


def key_for(symbol: str) -> bytes:
    # Exemple: BINANCE|BTCUSDT
    return f"{EXCHANGE_NAME}|{symbol.replace('/', '')}".encode()


def headers_for(msg_type: str, symbol: str):
    # headers en bytes (aiokafka)
    return [
        ("source", b"collector-ws"),
        ("exchange", EXCHANGE_NAME.encode()),
        ("symbol", symbol.replace("/", "").encode()),
        ("type", msg_type.encode()),
        ("schema", f"{msg_type}_v1".encode()),
    ]


def parse_timeframe(s: str) -> int:
    s = s.strip().lower()
    if s.endswith("ms"): return int(s[:-2])
    if s.endswith("s"):  return int(s[:-1]) * 1000
    if s.endswith("m"):  return int(s[:-1]) * 60_000
    if s.endswith("h"):  return int(s[:-1]) * 3_600_000
    if s.endswith("d"):  return int(s[:-1]) * 86_400_000
    raise ValueError(f"Bad timeframe: {s}")


def floor_ts(ts_ms: int, tf_ms: int) -> int:
    return ts_ms - (ts_ms % tf_ms)


def bar_key(exchange: str, symbol: str, tf_label: str, window_start: int) -> bytes:
    # EXCHANGE|BTCUSDT|1m|<window_start_ms>
    return f"{exchange}|{symbol.replace('/','')}|{tf_label}|{window_start}".encode()


def headers_for_bar(exchange: str, symbol: str, tf_label: str, closed: bool):
    return [
        ("source", b"collector-ws"),
        ("type", b"bar"),
        ("exchange", exchange.encode()),
        ("symbol", symbol.replace("/", "").encode()),
        ("timeframe", tf_label.encode()),
        ("closed", b"true" if closed else b"false"),
        ("schema", b"bar_v1"),
    ]


# ----------------------- in-process aggregator -----------------------
class InProcessAggregator:
    """
    Agrège les trades reçus (par symbole) en bougies multi-timeframes,
    et publie dans crypto.aggregated.<tf> uniquement à la fermeture de la fenêtre.
    """
    def __init__(self, producer: AIOKafkaProducer, output_prefix: str, timeframes: list[str],
                 grace_ms: int = 2000, sweep_sec: float = 1.0):
        self.producer = producer
        self.output_prefix = output_prefix  # ex: "crypto.aggregated."
        self.tf_labels = timeframes
        self.tf_ms = {lbl: parse_timeframe(lbl) for lbl in self.tf_labels}
        self.grace_ms = grace_ms
        self.sweep_interval = sweep_sec
        # (exchange, symbol, tf_label) -> bar dict
        self.windows: dict[tuple[str, str, str], dict] = {}
        self._stop = asyncio.Event()

    async def stop(self):
        self._stop.set()

    async def sweeper(self):
        while not self._stop.is_set():
            await asyncio.sleep(self.sweep_interval)
            now = int(time.time() * 1000)
            to_close = []
            for key, bar in list(self.windows.items()):
                tf_ms = self.tf_ms[bar["timeframe"]]
                window_end = bar["window_start"] + tf_ms
                if not bar["closed"] and now >= window_end + self.grace_ms:
                    to_close.append(key)
            for key in to_close:
                bar = self.windows.get(key)
                if bar and not bar["closed"]:
                    await self._emit_bar(bar, closed=True)

    async def on_trade(self, exchange: str, symbol: str, ts: int, price: float, amount: float):
        # met à jour toutes les TF pour ce symbole
        for tf_label, tf_ms in self.tf_ms.items():
            win_start = floor_ts(ts, tf_ms)
            key = (exchange, symbol, tf_label)
            bar = self.windows.get(key)

            if (bar is None) or (bar["window_start"] != win_start):
                # fermer l'ancienne fenêtre si elle existe
                if bar is not None and not bar["closed"]:
                    await self._emit_bar(bar, closed=True)

                # ouvrir une nouvelle fenêtre (pas d'émission incrémentale)
                bar = {
                    "type": "bar",
                    "exchange": exchange,
                    "symbol": symbol,
                    "timeframe": tf_label,
                    "window_start": win_start,
                    "open": price,
                    "high": price,
                    "low": price,
                    "close": price,
                    "volume": amount,  # volume base
                    "closed": False,
                    "last_update": ts,
                }
                self.windows[key] = bar
            else:
                # mise à jour de la fenêtre courante (pas d'émission incrémentale)
                if price > bar["high"]:
                    bar["high"] = price
                if price < bar["low"]:
                    bar["low"] = price
                bar["close"] = price
                bar["volume"] += amount
                bar["last_update"] = ts

    async def _emit_bar(self, bar: dict, closed: bool):
        out = dict(bar)
        out["closed"] = closed
        topic = f"{self.output_prefix}{bar['timeframe']}"
        keyb = bar_key(bar["exchange"], bar["symbol"], bar["timeframe"], bar["window_start"])
        headers = headers_for_bar(bar["exchange"], bar["symbol"], bar["timeframe"], closed)
        await self.producer.send_and_wait(topic, dumps(out), key=keyb, headers=headers)
        if closed:
            bar["closed"] = True


# --------------------------- workers ---------------------------------
async def watch_trades(symbol: str, ex: ccxtpro.binance, producer: AIOKafkaProducer,
                       topic_raw: str, aggregator: InProcessAggregator | None):
    last_id = None
    while True:
        try:
            trades = await ex.watch_trades(symbol)
            if not trades:
                continue
            t = trades[-1]  # cache ccxt.pro : on prend la plus récente
            tid = t.get("id")
            if tid and tid == last_id:
                continue
            last_id = tid

            ts = int(t["timestamp"]) if t.get("timestamp") is not None else int(time.time() * 1000)
            price = float(t["price"])
            amount = float(t["amount"])

            # 1) publier le trade brut
            payload = {
                "type": "trade",
                "exchange": EXCHANGE_NAME,
                "symbol": symbol,
                "event_ts": ts,
                "price": price,
                "amount": amount,
                "side": t.get("side"),
                "trade_id": tid,
            }
            await producer.send_and_wait(
                topic_raw,
                dumps(payload),
                key=key_for(symbol),
                headers=headers_for("trade", symbol),
            )

            # 2) alimenter l'agrégateur in-process (si activé)
            if aggregator is not None:
                await aggregator.on_trade(EXCHANGE_NAME, symbol, ts, price, amount)

        except Exception as e:
            print(f"[trades:{symbol}] {e}")
            await asyncio.sleep(3)  # petit backoff et on reprend


async def watch_ticker(symbol: str, ex: ccxtpro.binance, producer: AIOKafkaProducer, topic: str):
    while True:
        try:
            tk = await ex.watch_ticker(symbol)
            payload = {
                "type": "ticker",
                "exchange": EXCHANGE_NAME,
                "symbol": symbol,
                "event_ts": int(tk["timestamp"]) if tk.get("timestamp") is not None else None,
                "bid": float(tk["bid"]) if tk.get("bid") is not None else None,
                "ask": float(tk["ask"]) if tk.get("ask") is not None else None,
                "last": float(tk["last"]) if tk.get("last") is not None else None,
                "baseVolume": float(tk["baseVolume"]) if tk.get("baseVolume") is not None else None,
                "quoteVolume": float(tk["quoteVolume"]) if tk.get("quoteVolume") is not None else None,
            }
            await producer.send_and_wait(
                topic,
                dumps(payload),
                key=key_for(symbol),
                headers=headers_for("ticker", symbol),
            )
        except Exception as e:
            print(f"[ticker:{symbol}] {e}")
            await asyncio.sleep(3)


# ---------------------------- main -----------------------------------
async def main():
    # --- config via env
    symbols_env = os.getenv("SYMBOLS", "BTC/USDT,ETH/USDT")
    symbols = [s.strip().upper().replace(" ", "") for s in symbols_env.split(",") if s.strip()]
    enable_trades = os.getenv("ENABLE_TRADES", "true").lower() == "true"
    enable_ticker = os.getenv("ENABLE_TICKER", "true").lower() == "true"

    # agrégation in-process on/off
    enable_agg = os.getenv("ENABLE_AGGREGATION", "true").lower() == "true"
    output_prefix = os.getenv("OUTPUT_PREFIX", "crypto.aggregated.")
    timeframes = [t.strip() for t in os.getenv("TIMEFRAMES", "5s,1m,15m,1h").split(",") if t.strip()]
    grace_ms = int(os.getenv("GRACE_MS", "2000"))
    sweep_sec = float(os.getenv("SWEEP_SEC", "1.0"))

    kafka_bootstrap = os.getenv("KAFKA_BROKERS", "localhost:9092")
    topic_trades = os.getenv("TOPIC_TRADES", "crypto.raw.trades")
    topic_ticker = os.getenv("TOPIC_TICKER", "crypto.ticker")
    client_id = os.getenv("KAFKA_CLIENT_ID", "collector-ws")

    # Par défaut -> gzip pour éviter Snappy
    compression = (os.getenv("KAFKA_COMPRESSION", "gzip") or "").lower()
    if compression in ("", "none"):
        compression = None

    default_type = os.getenv("BINANCE_DEFAULT_TYPE", "spot")  # "spot" | "future" | "delivery"
    api_key = os.getenv("BINANCE_API_KEY")
    api_secret = os.getenv("BINANCE_SECRET_KEY")

    print(
        f"[startup] symbols={symbols} trades={enable_trades} ticker={enable_ticker} agg={enable_agg} "
        f"timeframes={timeframes} kafka={kafka_bootstrap} topics=({topic_trades},{topic_ticker},{output_prefix}*) "
        f"compression={compression or 'none'} default_type={default_type}"
    )

    # --- exchange
    ex = ccxtpro.binance(
        {
            "enableRateLimit": True,
            "options": {"defaultType": default_type},
        }
    )
    if api_key and api_secret:
        ex.apiKey = api_key
        ex.secret = api_secret

    try:
        await ex.load_markets()
    except Exception as e:
        print(f"[startup] load_markets error: {e}")

    # --- kafka producer
    producer = AIOKafkaProducer(
        bootstrap_servers=kafka_bootstrap,
        client_id=client_id,
        value_serializer=lambda v: v,  # dumps() renvoie bytes
        key_serializer=lambda k: k,
        linger_ms=10,
        compression_type=compression,  # gzip par défaut
        acks="all",
        enable_idempotence=True,
        max_request_size=1_048_576,
    )
    await producer.start()

    # --- aggregator in-process
    aggregator = InProcessAggregator(producer, output_prefix, timeframes, grace_ms, sweep_sec) if enable_agg else None

    # --- graceful shutdown
    stop = asyncio.Event()

    def _shutdown():
        stop.set()

    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, _shutdown)
        except NotImplementedError:
            pass

    # --- tasks
    tasks = []
    if aggregator is not None:
        tasks.append(asyncio.create_task(aggregator.sweeper()))
    for s in symbols:
        if enable_trades:
            tasks.append(asyncio.create_task(watch_trades(s, ex, producer, topic_trades, aggregator)))
        if enable_ticker:
            tasks.append(asyncio.create_task(watch_ticker(s, ex, producer, topic_ticker)))

    print(f"[startup] started {len(tasks)} task(s) for {len(symbols)} symbol(s).")

    try:
        await stop.wait()
    finally:
        for t in tasks:
            t.cancel()
        try:
            await asyncio.gather(*tasks, return_exceptions=True)
        finally:
            if aggregator is not None:
                await aggregator.stop()
            await producer.stop()
            await ex.close()
        print("[shutdown] collector+aggregator stopped cleanly.")


if __name__ == "__main__":
    asyncio.run(main())
