import os, asyncio, signal
import orjson

# uvloop facultatif (déjà dans tes deps)
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


# --------------------------- workers ---------------------------------
async def watch_trades(symbol: str, ex: ccxtpro.binance, producer: AIOKafkaProducer, topic: str):
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

            payload = {
                "type": "trade",
                "exchange": EXCHANGE_NAME,
                "symbol": symbol,
                "event_ts": int(t["timestamp"]) if t.get("timestamp") is not None else None,
                "price": float(t["price"]),
                "amount": float(t["amount"]),
                "side": t.get("side"),
                "trade_id": tid,
            }
            await producer.send_and_wait(
                topic,
                dumps(payload),
                key=key_for(symbol),
                headers=headers_for("trade", symbol),
            )
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

    kafka_bootstrap = os.getenv("KAFKA_BROKERS", "localhost:9092")
    topic_trades = os.getenv("TOPIC_TRADES", "crypto.raw.trades")
    topic_ticker = os.getenv("TOPIC_TICKER", "crypto.ticker")
    client_id = os.getenv("KAFKA_CLIENT_ID", "collector-ws")

    # Par défaut -> gzip pour éviter les soucis de lib snappy
    compression = (os.getenv("KAFKA_COMPRESSION", "gzip") or "").lower()
    if compression in ("", "none"):
        compression = None

    default_type = os.getenv("BINANCE_DEFAULT_TYPE", "spot")  # "spot" | "future" | "delivery"
    api_key = os.getenv("BINANCE_API_KEY")
    api_secret = os.getenv("BINANCE_SECRET_KEY")

    # --- log config de démarrage
    print(
        f"[startup] symbols={symbols} trades={enable_trades} ticker={enable_ticker} "
        f"kafka_bootstrap={kafka_bootstrap} topics=({topic_trades},{topic_ticker}) "
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

    # Charger les marchés pour valider les symboles (meilleur feedback si erreur)
    try:
        await ex.load_markets()
    except Exception as e:
        print(f"[startup] load_markets error: {e}")

    # --- kafka producer
    producer = AIOKafkaProducer(
        bootstrap_servers=kafka_bootstrap,
        client_id=client_id,
        value_serializer=lambda v: v,  # dumps() renvoie déjà bytes
        key_serializer=lambda k: k,
        linger_ms=10,
        compression_type=compression,  # gzip par défaut (évite la dépendance snappy)
        acks="all",
        enable_idempotence=True,
        max_request_size=1_048_576,
    )
    await producer.start()

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
    for s in symbols:
        if enable_trades:
            tasks.append(asyncio.create_task(watch_trades(s, ex, producer, topic_trades)))
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
            await producer.stop()
            await ex.close()
        print("[shutdown] collector stopped cleanly.")


if __name__ == "__main__":
    asyncio.run(main())
