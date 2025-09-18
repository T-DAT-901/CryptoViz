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

# ----------------------- Récupération automatique des symboles -----------------------
async def get_all_binance_symbols(ex, quote_filter=None, min_volume_filter=None, max_symbols=None):
    """
    Récupère tous les symboles disponibles sur Binance avec filtres optionnels
    
    Args:
        ex: Instance ccxt exchange
        quote_filter: Liste des devises de cotation à inclure (ex: ['USDT', 'BTC'])
        min_volume_filter: Volume minimum en quote currency sur 24h
        max_symbols: Nombre maximum de symboles à retourner
    
    Returns:
        List des symboles filtrés
    """
    try:
        print("[symbols] Chargement des marchés Binance...")
        await ex.load_markets()
        
        print("[symbols] Récupération des tickers pour filtrage par volume...")
        tickers = await ex.fetch_tickers()
        
        all_symbols = []
        filtered_symbols = []
        quote_stats = {}
        
        for symbol, market in ex.markets.items():
            # Vérifier que le marché est actif et spot
            if not market.get('active', True):
                continue
                
            if market.get('type') != 'spot':
                continue
                
            all_symbols.append(symbol)
            
            # Statistiques par devise de cotation
            quote_currency = market.get('quote', 'UNKNOWN')
            quote_stats[quote_currency] = quote_stats.get(quote_currency, 0) + 1
            
            # Appliquer les filtres si spécifiés
            should_include = True
            
            # Filtre par devise de cotation
            if quote_filter:
                if quote_currency not in quote_filter:
                    should_include = False
                    continue
            
            # Filtre par volume minimum
            if min_volume_filter and should_include:
                ticker = tickers.get(symbol)
                if ticker and ticker.get('quoteVolume'):
                    if ticker['quoteVolume'] < min_volume_filter:
                        should_include = False
                else:
                    should_include = False  # Skip si pas de données de volume
            
            if should_include:
                filtered_symbols.append(symbol)
        
        print(f"[symbols] Total des marchés spot actifs: {len(all_symbols)}")
        print(f"[symbols] Symboles après filtrage: {len(filtered_symbols)}")
        
        # Afficher les statistiques par devise de cotation
        print("[symbols] Répartition par devise de cotation:")
        for quote, count in sorted(quote_stats.items(), key=lambda x: x[1], reverse=True)[:10]:
            print(f"   {quote}: {count} paires")
        
        # Trier les symboles filtrés par volume (si on a des tickers)
        if min_volume_filter and tickers:
            def get_volume(symbol):
                ticker = tickers.get(symbol, {})
                return ticker.get('quoteVolume', 0) or 0
            
            filtered_symbols.sort(key=get_volume, reverse=True)
            print(f"[symbols] Top 10 par volume: {filtered_symbols[:10]}")
        
        # Limiter le nombre de symboles si spécifié
        if max_symbols and len(filtered_symbols) > max_symbols:
            filtered_symbols = filtered_symbols[:max_symbols]
            print(f"[symbols] Limité à {max_symbols} symboles")
        
        return filtered_symbols
        
    except Exception as e:
        print(f"[symbols] Erreur lors de la récupération: {e}")
        # Fallback sur une liste de base
        fallback = [
            'BTC/USDT', 'ETH/USDT', 'BNB/USDT', 'ADA/USDT', 'XRP/USDT',
            'SOL/USDT', 'DOGE/USDT', 'DOT/USDT', 'MATIC/USDT', 'LTC/USDT'
        ]
        print(f"[symbols] Utilisation de la liste de fallback: {len(fallback)} symboles")
        return fallback

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
    error_count = 0
    max_errors = 5
    
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
            
            # Reset error count on success
            error_count = 0

        except Exception as e:
            error_count += 1
            if error_count <= 3:  # Éviter le spam de logs
                print(f"[trades:{symbol}] Erreur {error_count}/{max_errors}: {e}")
            
            if error_count >= max_errors:
                print(f"[trades:{symbol}] Trop d'erreurs, arrêt du worker")
                break
                
            await asyncio.sleep(min(error_count * 2, 30))  # Backoff progressif

async def watch_ticker(symbol: str, ex: ccxtpro.binance, producer: AIOKafkaProducer, topic: str):
    error_count = 0
    max_errors = 5
    
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
            
            error_count = 0
            
        except Exception as e:
            error_count += 1
            if error_count <= 3:
                print(f"[ticker:{symbol}] Erreur {error_count}/{max_errors}: {e}")
            
            if error_count >= max_errors:
                print(f"[ticker:{symbol}] Trop d'erreurs, arrêt du worker")
                break
                
            await asyncio.sleep(min(error_count * 2, 30))

# ---------------------------- main -----------------------------------
async def main():
    # --- Configuration des symboles ---
    # Par défaut, récupérer toutes les paires disponibles
    # Modifiez ces variables pour ajuster le filtrage
    
    # Filtrer par devises de cotation (None = toutes les devises)
    quote_currencies = None  # ou ['USDT', 'BTC', 'ETH'] pour filtrer
    
    # Volume minimum sur 24h (None = pas de filtre)
    min_volume = None  # ou 100000 pour 100k minimum
    
    # Nombre maximum de symboles (None = pas de limite)
    max_symbols = None  # ou 50 pour limiter à 50
    
    print(f"[config] Récupération de TOUTES les paires disponibles")
    print(f"[config] Filtres: quotes={quote_currencies}, min_volume={min_volume}, max={max_symbols}")

    # --- autres configs via env (optionnel) ---
    enable_trades = os.getenv("ENABLE_TRADES", "true").lower() == "true"
    enable_ticker = os.getenv("ENABLE_TICKER", "true").lower() == "true"  # Désactivé par défaut

    # agrégation in-process on/off
    enable_agg = os.getenv("ENABLE_AGGREGATION", "true").lower() == "true"
    output_prefix = os.getenv("OUTPUT_PREFIX", "crypto.aggregated.")
    timeframes = [t.strip() for t in os.getenv("TIMEFRAMES", "5s,1m,15m,1h").split(",") if t.strip()]
    grace_ms = int(os.getenv("GRACE_MS", "2000"))
    sweep_sec = float(os.getenv("SWEEP_SEC", "1.0"))

    kafka_bootstrap = os.getenv("KAFKA_BROKERS", "localhost:9092")
    topic_trades = os.getenv("TOPIC_TRADES", "crypto.raw.trades")
    topic_ticker = os.getenv("TOPIC_TICKER", "crypto.ticker")
    client_id = os.getenv("KAFKA_CLIENT_ID", "collector-all-pairs")

    # Par défaut -> lz4 pour de meilleures performances avec beaucoup de données
    compression = (os.getenv("KAFKA_COMPRESSION", "lz4") or "").lower()
    if compression in ("", "none"):
        compression = None

    default_type = os.getenv("BINANCE_DEFAULT_TYPE", "spot")
    api_key = os.getenv("BINANCE_API_KEY")
    api_secret = os.getenv("BINANCE_SECRET_KEY")

    # --- exchange ---
    ex = ccxtpro.binance({
        "enableRateLimit": True,
        "options": {"defaultType": default_type},
    })
    if api_key and api_secret:
        ex.apiKey = api_key
        ex.secret = api_secret

    # --- Récupération automatique des symboles ---
    symbols = await get_all_binance_symbols(
        ex, 
        quote_filter=quote_currencies,
        min_volume_filter=min_volume,
        max_symbols=max_symbols
    )
    
    if not symbols:
        print("[error] Aucun symbole trouvé, arrêt du programme")
        return

    print(f"[startup] Configuration finale:")
    print(f"  - {len(symbols)} symboles à surveiller")
    print(f"  - Features: trades={enable_trades}, ticker={enable_ticker}, agg={enable_agg}")
    print(f"  - Kafka: {kafka_bootstrap}, compression={compression or 'none'}")
    print(f"  - Timeframes: {timeframes}")

    # --- kafka producer ---
    producer = AIOKafkaProducer(
        bootstrap_servers=kafka_bootstrap,
        client_id=client_id,
        value_serializer=lambda v: v,  # dumps() renvoie bytes
        key_serializer=lambda k: k,
        linger_ms=50,  # Attendre un peu plus pour de gros batches
        compression_type=compression,
        acks="all",  # Requis avec enable_idempotence=True
        enable_idempotence=True,
        max_request_size=2_097_152,  # 2MB
    )
    await producer.start()
    print("[startup] Producer Kafka démarré")

    # --- aggregator in-process ---
    aggregator = InProcessAggregator(producer, output_prefix, timeframes, grace_ms, sweep_sec) if enable_agg else None

    # --- graceful shutdown ---
    stop = asyncio.Event()

    def _shutdown():
        print("[signal] Signal d'arrêt reçu...")
        stop.set()

    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, _shutdown)
        except NotImplementedError:
            pass

    # --- tasks ---
    tasks = []
    if aggregator is not None:
        tasks.append(asyncio.create_task(aggregator.sweeper()))
    
    # Limitation de concurrence pour éviter de surcharger l'API
    max_concurrent = min(len(symbols), 50)  # Max 50 connexions simultanées
    semaphore = asyncio.Semaphore(max_concurrent)
    
    async def limited_watch_trades(symbol):
        async with semaphore:
            await watch_trades(symbol, ex, producer, topic_trades, aggregator)
    
    for s in symbols:
        if enable_trades:
            tasks.append(asyncio.create_task(limited_watch_trades(s)))
        if enable_ticker:
            tasks.append(asyncio.create_task(watch_ticker(s, ex, producer, topic_ticker)))

    print(f"[startup] {len(tasks)} tâches démarrées pour {len(symbols)} symboles (max {max_concurrent} concurrentes)")
    print("[startup] Le collecteur démarre... Utilisez Ctrl+C pour arrêter proprement")

    try:
        await stop.wait()
    finally:
        print("[shutdown] Arrêt en cours...")
        for t in tasks:
            t.cancel()
        try:
            await asyncio.gather(*tasks, return_exceptions=True)
        finally:
            if aggregator is not None:
                await aggregator.stop()
            await producer.stop()
            await ex.close()
        print("[shutdown] Arrêt terminé proprement")

if __name__ == "__main__":
    asyncio.run(main())