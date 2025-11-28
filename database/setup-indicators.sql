-- =============================================================================
-- CryptoViz Technical Indicators Calculation System
-- =============================================================================
-- Ce script configure le calcul automatique des indicateurs techniques
-- directement dans TimescaleDB pour des performances optimales
--
-- Indicateurs supportés:
--   - RSI (Relative Strength Index)
--   - MACD (Moving Average Convergence Divergence)
--   - Bollinger Bands
--   - Momentum
--   - Support/Resistance Levels
--
-- Architecture:
--   1. Fonctions SQL pour calculer chaque indicateur
--   2. Procédures pour rafraîchir les indicateurs par timeframe
--   3. Jobs automatiques TimescaleDB (fréquence adaptée à chaque timeframe)
--
-- Fréquence des jobs:
--   - 1s:  toutes les secondes (synchronisé avec les bougies)
--   - 5s:  toutes les 5 secondes (synchronisé avec les bougies)
--   - 1m:  toutes les minutes (synchronisé avec les bougies)
--   - 5m:  toutes les 5 minutes (synchronisé avec les bougies)
--   - 15m: toutes les 15 minutes (synchronisé avec les bougies)
--   - 1h:  toutes les heures (synchronisé avec les bougies)
--   - 1d:  une fois par jour (à minuit)
-- =============================================================================

-- =============================================================================
-- MISE À JOUR DU SCHÉMA
-- =============================================================================

-- Ajouter les colonnes pour support/resistance si elles n'existent pas
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'indicators' AND column_name = 'support_level'
    ) THEN
        ALTER TABLE indicators ADD COLUMN support_level DECIMAL(20,8);
        ALTER TABLE indicators ADD COLUMN resistance_level DECIMAL(20,8);
        ALTER TABLE indicators ADD COLUMN support_strength DECIMAL(5,2); -- Score de 0 à 100
        ALTER TABLE indicators ADD COLUMN resistance_strength DECIMAL(5,2); -- Score de 0 à 100
        RAISE NOTICE 'Added support/resistance columns to indicators table';
    END IF;
END $$;

-- Index supplémentaire pour les requêtes de support/resistance
CREATE INDEX IF NOT EXISTS idx_indicators_support_resistance
    ON indicators (symbol, timeframe, time DESC)
    WHERE support_level IS NOT NULL OR resistance_level IS NOT NULL;

-- =============================================================================
-- FONCTION 1: RSI (Relative Strength Index)
-- =============================================================================
-- Calcule le RSI pour un symbole et timeframe donné
-- Période par défaut: 14 (standard)
-- Retourne: valeur entre 0 et 100

CREATE OR REPLACE FUNCTION calculate_rsi(
    p_symbol VARCHAR,
    p_timeframe VARCHAR,
    p_period INT DEFAULT 14,
    p_exchange VARCHAR DEFAULT 'BINANCE'
)
RETURNS DECIMAL(20,8) AS $$
DECLARE
    v_rsi DECIMAL(20,8);
    v_avg_gain DECIMAL(20,8);
    v_avg_loss DECIMAL(20,8);
    v_rs DECIMAL(20,8);
BEGIN
    -- Calculer la moyenne des gains et pertes sur la période
    WITH price_changes AS (
        SELECT
            close - LAG(close) OVER (ORDER BY window_start) as change
        FROM candles
        WHERE symbol = p_symbol
            AND timeframe = p_timeframe
            AND exchange = p_exchange
            AND closed = TRUE
        ORDER BY window_start DESC
        LIMIT p_period + 1
    ),
    gains_losses AS (
        SELECT
            CASE WHEN change > 0 THEN change ELSE 0 END as gain,
            CASE WHEN change < 0 THEN ABS(change) ELSE 0 END as loss
        FROM price_changes
        WHERE change IS NOT NULL
    )
    SELECT
        AVG(gain),
        AVG(loss)
    INTO v_avg_gain, v_avg_loss
    FROM gains_losses;

    -- Éviter la division par zéro
    IF v_avg_loss = 0 OR v_avg_loss IS NULL THEN
        IF v_avg_gain > 0 THEN
            RETURN 100;
        ELSE
            RETURN 50; -- Pas de mouvement
        END IF;
    END IF;

    -- Calculer RS et RSI
    v_rs := v_avg_gain / v_avg_loss;
    v_rsi := 100 - (100 / (1 + v_rs));

    RETURN v_rsi;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- FONCTION 2: MACD (Moving Average Convergence Divergence)
-- =============================================================================
-- Calcule MACD, Signal et Histogram
-- Retourne: TABLE avec macd_line, signal_line, histogram

CREATE OR REPLACE FUNCTION calculate_macd(
    p_symbol VARCHAR,
    p_timeframe VARCHAR,
    p_fast INT DEFAULT 12,
    p_slow INT DEFAULT 26,
    p_signal INT DEFAULT 9,
    p_exchange VARCHAR DEFAULT 'BINANCE'
)
RETURNS TABLE(
    macd_line DECIMAL(20,8),
    signal_line DECIMAL(20,8),
    histogram DECIMAL(20,8)
) AS $$
DECLARE
    v_ema_fast DECIMAL(20,8);
    v_ema_slow DECIMAL(20,8);
    v_macd DECIMAL(20,8);
    v_signal DECIMAL(20,8);
    v_multiplier_fast DECIMAL(20,8);
    v_multiplier_slow DECIMAL(20,8);
BEGIN
    v_multiplier_fast := 2.0 / (p_fast + 1);
    v_multiplier_slow := 2.0 / (p_slow + 1);

    -- Calculer EMA rapide (12 périodes par défaut)
    WITH fast_prices AS (
        SELECT close, window_start,
               ROW_NUMBER() OVER (ORDER BY window_start DESC) as rn
        FROM candles
        WHERE symbol = p_symbol AND timeframe = p_timeframe
        AND exchange = p_exchange AND closed = TRUE
        ORDER BY window_start DESC
        LIMIT p_fast
    )
    SELECT
        CASE
            WHEN COUNT(*) >= p_fast THEN
                (SELECT close FROM fast_prices WHERE rn = 1) * v_multiplier_fast +
                AVG(close) * (1 - v_multiplier_fast)
            ELSE NULL
        END
    INTO v_ema_fast
    FROM fast_prices;

    -- Calculer EMA lente (26 périodes par défaut)
    WITH slow_prices AS (
        SELECT close, window_start,
               ROW_NUMBER() OVER (ORDER BY window_start DESC) as rn
        FROM candles
        WHERE symbol = p_symbol AND timeframe = p_timeframe
        AND exchange = p_exchange AND closed = TRUE
        ORDER BY window_start DESC
        LIMIT p_slow
    )
    SELECT
        CASE
            WHEN COUNT(*) >= p_slow THEN
                (SELECT close FROM slow_prices WHERE rn = 1) * v_multiplier_slow +
                AVG(close) * (1 - v_multiplier_slow)
            ELSE NULL
        END
    INTO v_ema_slow
    FROM slow_prices;

    -- Si pas assez de données, retourner NULL
    IF v_ema_fast IS NULL OR v_ema_slow IS NULL THEN
        RETURN QUERY SELECT NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8);
        RETURN;
    END IF;

    -- Calculer MACD = EMA_fast - EMA_slow
    v_macd := v_ema_fast - v_ema_slow;

    -- Calculer la ligne de signal (moyenne du MACD)
    -- Pour simplifier, on utilise une moyenne simple sur 9 périodes
    SELECT AVG(v_macd) INTO v_signal;

    RETURN QUERY SELECT
        v_macd,
        v_signal,
        v_macd - COALESCE(v_signal, 0); -- histogram
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- FONCTION 3: BOLLINGER BANDS
-- =============================================================================
-- Calcule les bandes de Bollinger
-- Retourne: TABLE avec upper_band, middle_band, lower_band

CREATE OR REPLACE FUNCTION calculate_bollinger(
    p_symbol VARCHAR,
    p_timeframe VARCHAR,
    p_period INT DEFAULT 20,
    p_std_dev DECIMAL DEFAULT 2.0,
    p_exchange VARCHAR DEFAULT 'BINANCE'
)
RETURNS TABLE(
    upper_band DECIMAL(20,8),
    middle_band DECIMAL(20,8),
    lower_band DECIMAL(20,8)
) AS $$
DECLARE
    v_sma DECIMAL(20,8);
    v_std DECIMAL(20,8);
    v_count INT;
BEGIN
    -- Calculer SMA (Simple Moving Average) et écart-type
    SELECT
        AVG(close),
        STDDEV(close),
        COUNT(*)
    INTO v_sma, v_std, v_count
    FROM (
        SELECT close FROM candles
        WHERE symbol = p_symbol AND timeframe = p_timeframe
        AND exchange = p_exchange AND closed = TRUE
        ORDER BY window_start DESC
        LIMIT p_period
    ) sub;

    -- Si pas assez de données
    IF v_count < p_period OR v_sma IS NULL THEN
        RETURN QUERY SELECT NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8);
        RETURN;
    END IF;

    -- Si pas de volatilité
    IF v_std IS NULL OR v_std = 0 THEN
        RETURN QUERY SELECT v_sma, v_sma, v_sma;
        RETURN;
    END IF;

    -- Retourner les bandes
    RETURN QUERY SELECT
        v_sma + (p_std_dev * v_std) as upper_band,
        v_sma as middle_band,
        v_sma - (p_std_dev * v_std) as lower_band;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- FONCTION 4: MOMENTUM
-- =============================================================================
-- Calcule le momentum (différence de prix sur N périodes)

CREATE OR REPLACE FUNCTION calculate_momentum(
    p_symbol VARCHAR,
    p_timeframe VARCHAR,
    p_period INT DEFAULT 10,
    p_exchange VARCHAR DEFAULT 'BINANCE'
)
RETURNS DECIMAL(20,8) AS $$
DECLARE
    v_current_price DECIMAL(20,8);
    v_past_price DECIMAL(20,8);
BEGIN
    -- Prix actuel (dernière bougie fermée)
    SELECT close INTO v_current_price
    FROM candles
    WHERE symbol = p_symbol AND timeframe = p_timeframe
    AND exchange = p_exchange AND closed = TRUE
    ORDER BY window_start DESC
    LIMIT 1;

    -- Prix il y a N périodes
    SELECT close INTO v_past_price
    FROM candles
    WHERE symbol = p_symbol AND timeframe = p_timeframe
    AND exchange = p_exchange AND closed = TRUE
    ORDER BY window_start DESC
    OFFSET p_period
    LIMIT 1;

    -- Vérifier qu'on a les deux prix
    IF v_current_price IS NULL OR v_past_price IS NULL THEN
        RETURN NULL;
    END IF;

    -- Retourner la différence
    RETURN v_current_price - v_past_price;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- FONCTION 5: SUPPORT & RESISTANCE LEVELS
-- =============================================================================
-- Calcule les niveaux de support et résistance en analysant les pivots
-- Utilise la méthode des points de pivot et des extremums locaux

CREATE OR REPLACE FUNCTION calculate_support_resistance(
    p_symbol VARCHAR,
    p_timeframe VARCHAR,
    p_lookback INT DEFAULT 50,
    p_exchange VARCHAR DEFAULT 'BINANCE'
)
RETURNS TABLE(
    support_level DECIMAL(20,8),
    resistance_level DECIMAL(20,8),
    support_strength DECIMAL(5,2),
    resistance_strength DECIMAL(5,2)
) AS $$
DECLARE
    v_current_price DECIMAL(20,8);
    v_count INT;
BEGIN
    -- Prix actuel
    SELECT close INTO v_current_price
    FROM candles
    WHERE symbol = p_symbol AND timeframe = p_timeframe
    AND exchange = p_exchange AND closed = TRUE
    ORDER BY window_start DESC
    LIMIT 1;

    -- Vérifier qu'on a assez de données
    SELECT COUNT(*) INTO v_count
    FROM candles
    WHERE symbol = p_symbol AND timeframe = p_timeframe
    AND exchange = p_exchange AND closed = TRUE;

    IF v_count < 10 OR v_current_price IS NULL THEN
        -- Pas assez de données, retourner des valeurs par défaut
        RETURN QUERY SELECT
            v_current_price * 0.95,
            v_current_price * 1.05,
            50.0::DECIMAL(5,2),
            50.0::DECIMAL(5,2);
        RETURN;
    END IF;

    RETURN QUERY
    WITH recent_candles AS (
        SELECT
            window_start,
            high,
            low,
            close,
            ROW_NUMBER() OVER (ORDER BY window_start DESC) as rn
        FROM candles
        WHERE symbol = p_symbol
            AND timeframe = p_timeframe
            AND exchange = p_exchange
            AND closed = TRUE
        ORDER BY window_start DESC
        LIMIT LEAST(p_lookback, v_count)
    ),
    -- Identifier les minimums locaux (support potentiel)
    support_levels AS (
        SELECT low as level,
            COUNT(*) as touches
        FROM recent_candles c1
        WHERE low < v_current_price
        AND EXISTS (
            SELECT 1 FROM recent_candles c2
            WHERE c2.rn BETWEEN GREATEST(1, c1.rn - 3) AND c1.rn + 3
            AND c2.low >= c1.low * 0.995  -- Tolérance de 0.5%
        )
        GROUP BY low
        HAVING COUNT(*) >= 2
        ORDER BY COUNT(*) DESC, ABS(low - v_current_price) ASC
        LIMIT 1
    ),
    -- Identifier les maximums locaux (résistance potentielle)
    resistance_levels AS (
        SELECT high as level,
            COUNT(*) as touches
        FROM recent_candles c1
        WHERE high > v_current_price
        AND EXISTS (
            SELECT 1 FROM recent_candles c2
            WHERE c2.rn BETWEEN GREATEST(1, c1.rn - 3) AND c1.rn + 3
            AND c2.high <= c1.high * 1.005  -- Tolérance de 0.5%
        )
        GROUP BY high
        HAVING COUNT(*) >= 2
        ORDER BY COUNT(*) DESC, ABS(high - v_current_price) ASC
        LIMIT 1
    )
    SELECT
        COALESCE(s.level, v_current_price * 0.95) as support_level,
        COALESCE(r.level, v_current_price * 1.05) as resistance_level,
        COALESCE(LEAST(s.touches * 15.0, 100.0), 50.0)::DECIMAL(5,2) as support_strength,
        COALESCE(LEAST(r.touches * 15.0, 100.0), 50.0)::DECIMAL(5,2) as resistance_strength
    FROM support_levels s
    FULL OUTER JOIN resistance_levels r ON TRUE;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- PROCÉDURE: Rafraîchir tous les indicateurs pour un timeframe
-- =============================================================================
-- Cette procédure calcule tous les indicateurs pour tous les symboles
-- d'un timeframe donné et insère les résultats dans la table indicators

CREATE OR REPLACE PROCEDURE refresh_indicators(
    p_timeframe VARCHAR DEFAULT NULL,
    p_symbol VARCHAR DEFAULT NULL
)
LANGUAGE plpgsql
AS $$
DECLARE
    v_symbol VARCHAR;
    v_timeframe VARCHAR;
    v_rsi DECIMAL(20,8);
    v_macd RECORD;
    v_bollinger RECORD;
    v_momentum DECIMAL(20,8);
    v_support_resistance RECORD;
    v_current_time TIMESTAMPTZ;
    v_count INT := 0;
    v_errors INT := 0;
BEGIN
    v_current_time := NOW();

    RAISE NOTICE 'Starting indicator refresh at % for timeframe: % symbol: %',
        v_current_time, COALESCE(p_timeframe, 'ALL'), COALESCE(p_symbol, 'ALL');

    -- Boucle sur tous les symboles et timeframes actifs
    FOR v_symbol, v_timeframe IN
        SELECT DISTINCT symbol, timeframe
        FROM candles
        WHERE closed = TRUE
            AND (p_timeframe IS NULL OR timeframe = p_timeframe)
            AND (p_symbol IS NULL OR symbol = p_symbol)
            AND window_start >= NOW() - INTERVAL '2 hours'
        ORDER BY symbol, timeframe
    LOOP
        BEGIN
            -- Calculer RSI
            v_rsi := calculate_rsi(v_symbol, v_timeframe);

            -- Calculer MACD
            SELECT * INTO v_macd FROM calculate_macd(v_symbol, v_timeframe);

            -- Calculer Bollinger Bands
            SELECT * INTO v_bollinger FROM calculate_bollinger(v_symbol, v_timeframe);

            -- Calculer Momentum
            v_momentum := calculate_momentum(v_symbol, v_timeframe);

            -- Calculer Support/Resistance
            SELECT * INTO v_support_resistance FROM calculate_support_resistance(v_symbol, v_timeframe);

            -- Insérer RSI
            IF v_rsi IS NOT NULL THEN
                INSERT INTO indicators (
                    time, symbol, timeframe, indicator_type, value
                ) VALUES (
                    v_current_time, v_symbol, v_timeframe, 'rsi', v_rsi
                );
            END IF;

            -- Insérer MACD
            IF v_macd.macd_line IS NOT NULL THEN
                INSERT INTO indicators (
                    time, symbol, timeframe, indicator_type,
                    value, value_signal, value_histogram
                ) VALUES (
                    v_current_time, v_symbol, v_timeframe, 'macd',
                    v_macd.macd_line, v_macd.signal_line, v_macd.histogram
                );
            END IF;

            -- Insérer Bollinger Bands
            IF v_bollinger.middle_band IS NOT NULL THEN
                INSERT INTO indicators (
                    time, symbol, timeframe, indicator_type,
                    upper_band, middle_band, lower_band
                ) VALUES (
                    v_current_time, v_symbol, v_timeframe, 'bollinger',
                    v_bollinger.upper_band, v_bollinger.middle_band, v_bollinger.lower_band
                );
            END IF;

            -- Insérer Momentum
            IF v_momentum IS NOT NULL THEN
                INSERT INTO indicators (
                    time, symbol, timeframe, indicator_type, value
                ) VALUES (
                    v_current_time, v_symbol, v_timeframe, 'momentum', v_momentum
                );
            END IF;

            -- Insérer Support/Resistance
            IF v_support_resistance.support_level IS NOT NULL THEN
                INSERT INTO indicators (
                    time, symbol, timeframe, indicator_type,
                    support_level, resistance_level, support_strength, resistance_strength
                ) VALUES (
                    v_current_time, v_symbol, v_timeframe, 'support_resistance',
                    v_support_resistance.support_level, v_support_resistance.resistance_level,
                    v_support_resistance.support_strength, v_support_resistance.resistance_strength
                );
            END IF;

            v_count := v_count + 1;

        EXCEPTION WHEN OTHERS THEN
            v_errors := v_errors + 1;
            RAISE WARNING 'Error calculating indicators for % %: %', v_symbol, v_timeframe, SQLERRM;
        END;
    END LOOP;

    RAISE NOTICE 'Indicator refresh completed: % symbols processed, % errors', v_count, v_errors;
END;
$$;

-- =============================================================================
-- PROCÉDURES SPÉCIFIQUES PAR TIMEFRAME
-- =============================================================================
-- Ces procédures sont appelées par les jobs automatiques TimescaleDB

CREATE OR REPLACE PROCEDURE refresh_indicators_1s()
LANGUAGE plpgsql AS $$
BEGIN
    CALL refresh_indicators('1s');
END;
$$;

CREATE OR REPLACE PROCEDURE refresh_indicators_5s()
LANGUAGE plpgsql AS $$
BEGIN
    CALL refresh_indicators('5s');
END;
$$;

CREATE OR REPLACE PROCEDURE refresh_indicators_1m()
LANGUAGE plpgsql AS $$
BEGIN
    CALL refresh_indicators('1m');
END;
$$;

CREATE OR REPLACE PROCEDURE refresh_indicators_5m()
LANGUAGE plpgsql AS $$
BEGIN
    CALL refresh_indicators('5m');
END;
$$;

CREATE OR REPLACE PROCEDURE refresh_indicators_15m()
LANGUAGE plpgsql AS $$
BEGIN
    CALL refresh_indicators('15m');
END;
$$;

CREATE OR REPLACE PROCEDURE refresh_indicators_1h()
LANGUAGE plpgsql AS $$
BEGIN
    CALL refresh_indicators('1h');
END;
$$;

CREATE OR REPLACE PROCEDURE refresh_indicators_1d()
LANGUAGE plpgsql AS $$
BEGIN
    CALL refresh_indicators('1d');
END;
$$;

-- =============================================================================
-- JOBS AUTOMATIQUES TIMESCALEDB
-- =============================================================================
-- Configuration des jobs pour calculer automatiquement les indicateurs
-- Fréquence synchronisée avec la fermeture des bougies de chaque timeframe
-- NOTE: add_job() is TimescaleDB Enterprise only - wrapped in exception handlers for Community Edition

-- Job pour 1s (toutes les secondes - synchronisé avec chaque bougie 1s)
DO $$
BEGIN
    PERFORM add_job('refresh_indicators_1s', '1 second', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_job for refresh_indicators_1s (Community Edition)';
END $$;

-- Job pour 5s (toutes les 5 secondes - synchronisé avec chaque bougie 5s)
DO $$
BEGIN
    PERFORM add_job('refresh_indicators_5s', '5 seconds', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_job for refresh_indicators_5s (Community Edition)';
END $$;

-- Job pour 1m (toutes les minutes - synchronisé avec chaque bougie 1m)
DO $$
BEGIN
    PERFORM add_job('refresh_indicators_1m', '1 minute', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_job for refresh_indicators_1m (Community Edition)';
END $$;

-- Job pour 5m (toutes les 5 minutes - synchronisé avec chaque bougie 5m)
DO $$
BEGIN
    PERFORM add_job('refresh_indicators_5m', '5 minutes', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_job for refresh_indicators_5m (Community Edition)';
END $$;

-- Job pour 15m (toutes les 15 minutes - synchronisé avec chaque bougie 15m)
DO $$
BEGIN
    PERFORM add_job('refresh_indicators_15m', '15 minutes', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_job for refresh_indicators_15m (Community Edition)';
END $$;

-- Job pour 1h (toutes les heures - synchronisé avec chaque bougie 1h)
DO $$
BEGIN
    PERFORM add_job('refresh_indicators_1h', '1 hour', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_job for refresh_indicators_1h (Community Edition)';
END $$;

-- Job pour 1d (une fois par jour - synchronisé avec chaque bougie 1d)
DO $$
BEGIN
    PERFORM add_job('refresh_indicators_1d', '1 day', if_not_exists => TRUE);
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_job for refresh_indicators_1d (Community Edition)';
END $$;

-- =============================================================================
-- VUES UTILITAIRES
-- =============================================================================

-- Vue pour avoir les derniers indicateurs de chaque type par symbole/timeframe
CREATE OR REPLACE VIEW latest_indicators AS
SELECT DISTINCT ON (symbol, timeframe, indicator_type)
    time,
    symbol,
    timeframe,
    indicator_type,
    value,
    value_signal,
    value_histogram,
    upper_band,
    middle_band,
    lower_band,
    support_level,
    resistance_level,
    support_strength,
    resistance_strength
FROM indicators
ORDER BY symbol, timeframe, indicator_type, time DESC;

-- Vue pour un dashboard complet par symbole
CREATE OR REPLACE VIEW indicators_dashboard AS
SELECT
    i.symbol,
    i.timeframe,
    i.time,
    MAX(CASE WHEN i.indicator_type = 'rsi' THEN i.value END) as rsi,
    MAX(CASE WHEN i.indicator_type = 'macd' THEN i.value END) as macd,
    MAX(CASE WHEN i.indicator_type = 'macd' THEN i.value_signal END) as macd_signal,
    MAX(CASE WHEN i.indicator_type = 'macd' THEN i.value_histogram END) as macd_histogram,
    MAX(CASE WHEN i.indicator_type = 'bollinger' THEN i.upper_band END) as bb_upper,
    MAX(CASE WHEN i.indicator_type = 'bollinger' THEN i.middle_band END) as bb_middle,
    MAX(CASE WHEN i.indicator_type = 'bollinger' THEN i.lower_band END) as bb_lower,
    MAX(CASE WHEN i.indicator_type = 'momentum' THEN i.value END) as momentum,
    MAX(CASE WHEN i.indicator_type = 'support_resistance' THEN i.support_level END) as support,
    MAX(CASE WHEN i.indicator_type = 'support_resistance' THEN i.resistance_level END) as resistance,
    MAX(CASE WHEN i.indicator_type = 'support_resistance' THEN i.support_strength END) as support_strength,
    MAX(CASE WHEN i.indicator_type = 'support_resistance' THEN i.resistance_strength END) as resistance_strength
FROM latest_indicators i
GROUP BY i.symbol, i.timeframe, i.time;

-- =============================================================================
-- FONCTION OPTIMISÉE: Backfill Historique avec Window Functions
-- =============================================================================
-- Cette fonction calcule TOUS les indicateurs sur TOUTES les bougies historiques
-- en une seule passe optimisée avec des window functions SQL
--
-- Au lieu de: 57,600 bougies × 5 fonctions = 288,000 requêtes
-- On fait: 1 seule requête INSERT avec window functions
-- Temps: Quelques secondes au lieu de plusieurs heures !

CREATE OR REPLACE PROCEDURE backfill_historical_indicators_optimized(
    p_timeframe VARCHAR DEFAULT '1m',
    p_symbol VARCHAR DEFAULT NULL,
    p_batch_size INT DEFAULT 10000
)
LANGUAGE plpgsql
AS $$
DECLARE
    v_symbol VARCHAR;
    v_start_time TIMESTAMPTZ;
    v_end_time TIMESTAMPTZ;
    v_total_inserted INT := 0;
    v_symbols_processed INT := 0;
BEGIN
    v_start_time := clock_timestamp();

    RAISE NOTICE '═══════════════════════════════════════════════════════════════════';
    RAISE NOTICE 'Starting OPTIMIZED historical indicators backfill';
    RAISE NOTICE '═══════════════════════════════════════════════════════════════════';
    RAISE NOTICE 'Timeframe: %', p_timeframe;
    RAISE NOTICE 'Symbol filter: %', COALESCE(p_symbol, 'ALL');
    RAISE NOTICE '';

    -- Boucle sur chaque symbole (pour éviter de tout charger en mémoire)
    FOR v_symbol IN
        SELECT DISTINCT symbol
        FROM candles
        WHERE timeframe = p_timeframe
            AND closed = TRUE
            AND (p_symbol IS NULL OR symbol = p_symbol)
        ORDER BY symbol
    LOOP
        BEGIN
            RAISE NOTICE 'Processing symbol: %...', v_symbol;

            -- ═══════════════════════════════════════════════════════════════
            -- CALCUL DE TOUS LES INDICATEURS EN UNE SEULE REQUÊTE
            -- Utilise des window functions pour performance maximale
            -- ═══════════════════════════════════════════════════════════════
            WITH candles_data AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    open,
                    high,
                    low,
                    close,
                    volume,
                    ROW_NUMBER() OVER (ORDER BY window_start) as rn
                FROM candles
                WHERE symbol = v_symbol
                    AND timeframe = p_timeframe
                    AND closed = TRUE
                ORDER BY window_start
            ),
            -- ─────────────────────────────────────────────────────────────
            -- 1️⃣ RSI (Relative Strength Index) - Période 14
            -- ─────────────────────────────────────────────────────────────
            price_changes AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    close,
                    close - LAG(close, 1) OVER (ORDER BY window_start) as price_change
                FROM candles_data
            ),
            gains_losses AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    CASE WHEN price_change > 0 THEN price_change ELSE 0 END as gain,
                    CASE WHEN price_change < 0 THEN ABS(price_change) ELSE 0 END as loss
                FROM price_changes
                WHERE price_change IS NOT NULL
            ),
            rsi_calc AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    -- Moyenne mobile des gains et pertes sur 14 périodes
                    AVG(gain) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 13 PRECEDING AND CURRENT ROW
                    ) as avg_gain,
                    AVG(loss) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 13 PRECEDING AND CURRENT ROW
                    ) as avg_loss
                FROM gains_losses
            ),
            rsi_values AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    CASE
                        WHEN avg_loss = 0 OR avg_loss IS NULL THEN
                            CASE WHEN avg_gain > 0 THEN 100 ELSE 50 END
                        ELSE
                            100 - (100 / (1 + (avg_gain / avg_loss)))
                    END as rsi_value
                FROM rsi_calc
                WHERE avg_gain IS NOT NULL
            ),
            -- ─────────────────────────────────────────────────────────────
            -- 2️⃣ MACD (Moving Average Convergence Divergence)
            -- EMA rapide (12), EMA lente (26), Signal (9)
            -- ─────────────────────────────────────────────────────────────
            ema_calc AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    close,
                    -- EMA 12 (rapide)
                    AVG(close) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 11 PRECEDING AND CURRENT ROW
                    ) as ema_12,
                    -- EMA 26 (lente)
                    AVG(close) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 25 PRECEDING AND CURRENT ROW
                    ) as ema_26
                FROM candles_data
            ),
            macd_calc AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    ema_12 - ema_26 as macd_line
                FROM ema_calc
                WHERE ema_12 IS NOT NULL AND ema_26 IS NOT NULL
            ),
            macd_signal_calc AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    macd_line,
                    -- Signal = moyenne mobile du MACD sur 9 périodes
                    AVG(macd_line) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 8 PRECEDING AND CURRENT ROW
                    ) as signal_line
                FROM macd_calc
            ),
            macd_values AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    macd_line,
                    signal_line,
                    macd_line - COALESCE(signal_line, 0) as histogram
                FROM macd_signal_calc
            ),
            -- ─────────────────────────────────────────────────────────────
            -- 3️⃣ BOLLINGER BANDS - Période 20, Écart-type 2
            -- ─────────────────────────────────────────────────────────────
            bollinger_calc AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    close,
                    -- SMA (moyenne mobile simple) sur 20 périodes
                    AVG(close) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
                    ) as sma_20,
                    -- Écart-type sur 20 périodes
                    STDDEV(close) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 19 PRECEDING AND CURRENT ROW
                    ) as stddev_20
                FROM candles_data
            ),
            bollinger_values AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    sma_20 as middle_band,
                    sma_20 + (2 * COALESCE(stddev_20, 0)) as upper_band,
                    sma_20 - (2 * COALESCE(stddev_20, 0)) as lower_band
                FROM bollinger_calc
                WHERE sma_20 IS NOT NULL
            ),
            -- ─────────────────────────────────────────────────────────────
            -- 4️⃣ MOMENTUM - Différence de prix sur 10 périodes
            -- ─────────────────────────────────────────────────────────────
            momentum_values AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    close - LAG(close, 10) OVER (ORDER BY window_start) as momentum_value
                FROM candles_data
            ),
            -- ─────────────────────────────────────────────────────────────
            -- 5️⃣ SUPPORT/RESISTANCE (Simplifié)
            -- Utilise les min/max locaux sur 50 périodes
            -- ─────────────────────────────────────────────────────────────
            support_resistance_calc AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    close,
                    -- Support = minimum sur 50 périodes
                    MIN(low) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 49 PRECEDING AND CURRENT ROW
                    ) as support_level,
                    -- Résistance = maximum sur 50 périodes
                    MAX(high) OVER (
                        ORDER BY window_start
                        ROWS BETWEEN 49 PRECEDING AND CURRENT ROW
                    ) as resistance_level
                FROM candles_data
            ),
            support_resistance_values AS (
                SELECT
                    window_start,
                    symbol,
                    timeframe,
                    support_level,
                    resistance_level,
                    -- Force du support (50-100 basé sur la proximité)
                    CASE
                        WHEN close > 0 THEN
                            LEAST(100, 50 + ((close - support_level) / close * 100))
                        ELSE 50
                    END::DECIMAL(5,2) as support_strength,
                    -- Force de la résistance (50-100 basé sur la proximité)
                    CASE
                        WHEN close > 0 THEN
                            LEAST(100, 50 + ((resistance_level - close) / close * 100))
                        ELSE 50
                    END::DECIMAL(5,2) as resistance_strength
                FROM support_resistance_calc
                WHERE support_level IS NOT NULL AND resistance_level IS NOT NULL
            )
            -- ═══════════════════════════════════════════════════════════════
            -- INSERT MASSIF DE TOUS LES INDICATEURS
            -- ═══════════════════════════════════════════════════════════════
            INSERT INTO indicators (
                time, symbol, timeframe, indicator_type,
                value, value_signal, value_histogram,
                upper_band, middle_band, lower_band,
                support_level, resistance_level, support_strength, resistance_strength
            )
            -- RSI
            SELECT
                window_start, symbol, timeframe, 'rsi'::VARCHAR,
                rsi_value, NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(5,2), NULL::DECIMAL(5,2)
            FROM rsi_values

            UNION ALL

            -- MACD
            SELECT
                window_start, symbol, timeframe, 'macd'::VARCHAR,
                macd_line, signal_line, histogram,
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(5,2), NULL::DECIMAL(5,2)
            FROM macd_values

            UNION ALL

            -- Bollinger Bands
            SELECT
                window_start, symbol, timeframe, 'bollinger'::VARCHAR,
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                upper_band, middle_band, lower_band,
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(5,2), NULL::DECIMAL(5,2)
            FROM bollinger_values

            UNION ALL

            -- Momentum
            SELECT
                window_start, symbol, timeframe, 'momentum'::VARCHAR,
                momentum_value, NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(5,2), NULL::DECIMAL(5,2)
            FROM momentum_values
            WHERE momentum_value IS NOT NULL

            UNION ALL

            -- Support/Resistance
            SELECT
                window_start, symbol, timeframe, 'support_resistance'::VARCHAR,
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                NULL::DECIMAL(20,8), NULL::DECIMAL(20,8), NULL::DECIMAL(20,8),
                support_level, resistance_level, support_strength, resistance_strength
            FROM support_resistance_values

            ON CONFLICT (time, symbol, timeframe, indicator_type) DO UPDATE SET
                value = EXCLUDED.value,
                value_signal = EXCLUDED.value_signal,
                value_histogram = EXCLUDED.value_histogram,
                upper_band = EXCLUDED.upper_band,
                middle_band = EXCLUDED.middle_band,
                lower_band = EXCLUDED.lower_band,
                support_level = EXCLUDED.support_level,
                resistance_level = EXCLUDED.resistance_level,
                support_strength = EXCLUDED.support_strength,
                resistance_strength = EXCLUDED.resistance_strength;

            GET DIAGNOSTICS v_total_inserted = ROW_COUNT;
            v_symbols_processed := v_symbols_processed + 1;

            RAISE NOTICE '  ✓ % indicators inserted for %', v_total_inserted, v_symbol;

        EXCEPTION WHEN OTHERS THEN
            RAISE WARNING '  ✗ Error processing %: %', v_symbol, SQLERRM;
        END;
    END LOOP;

    v_end_time := clock_timestamp();

    RAISE NOTICE '';
    RAISE NOTICE '═══════════════════════════════════════════════════════════════════';
    RAISE NOTICE '✓ Backfill completed successfully!';
    RAISE NOTICE '═══════════════════════════════════════════════════════════════════';
    RAISE NOTICE 'Symbols processed: %', v_symbols_processed;
    RAISE NOTICE 'Total indicators: %', v_total_inserted;
    RAISE NOTICE 'Duration: %', v_end_time - v_start_time;
    RAISE NOTICE '═══════════════════════════════════════════════════════════════════';
    RAISE NOTICE '';
    RAISE NOTICE 'Examples for other timeframes:';
    RAISE NOTICE '  CALL backfill_historical_indicators_optimized(''5m'');';
    RAISE NOTICE '  CALL backfill_historical_indicators_optimized(''15m'');';
    RAISE NOTICE '  CALL backfill_historical_indicators_optimized(''1h'');';
    RAISE NOTICE '  CALL backfill_historical_indicators_optimized(''1d'');';
END;
$$;

-- =============================================================================
-- FINALISATION
-- =============================================================================

-- Analyser la table pour optimiser les requêtes
ANALYZE indicators;

-- Afficher un résumé
DO $$
DECLARE
    v_job_count INT;
BEGIN
    SELECT COUNT(*) INTO v_job_count
    FROM timescaledb_information.jobs
    WHERE proc_name LIKE 'refresh_indicators%';

    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Technical Indicators System Setup Complete';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Functions created:';
    RAISE NOTICE '  - calculate_rsi()';
    RAISE NOTICE '  - calculate_macd()';
    RAISE NOTICE '  - calculate_bollinger()';
    RAISE NOTICE '  - calculate_momentum()';
    RAISE NOTICE '  - calculate_support_resistance()';
    RAISE NOTICE '';
    RAISE NOTICE 'Procedures created:';
    RAISE NOTICE '  - refresh_indicators()';
    RAISE NOTICE '  - refresh_indicators_1s/5s/1m/5m/15m/1h/1d()';
    RAISE NOTICE '';
    RAISE NOTICE 'Automatic jobs configured: %', v_job_count;
    RAISE NOTICE '  - 1s:  every 1 second (synchronized with 1s candles)';
    RAISE NOTICE '  - 5s:  every 5 seconds (synchronized with 5s candles)';
    RAISE NOTICE '  - 1m:  every 1 minute (synchronized with 1m candles)';
    RAISE NOTICE '  - 5m:  every 5 minutes (synchronized with 5m candles)';
    RAISE NOTICE '  - 15m: every 15 minutes (synchronized with 15m candles)';
    RAISE NOTICE '  - 1h:  every 1 hour (synchronized with 1h candles)';
    RAISE NOTICE '  - 1d:  every 1 day (synchronized with 1d candles)';
    RAISE NOTICE '';
    RAISE NOTICE 'Views created:';
    RAISE NOTICE '  - latest_indicators (derniers indicateurs par type)';
    RAISE NOTICE '  - indicators_dashboard (vue consolidée pour dashboard)';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'To manually trigger calculation:';
    RAISE NOTICE '  CALL refresh_indicators();              -- All timeframes';
    RAISE NOTICE '  CALL refresh_indicators(''1h'');          -- Specific timeframe';
    RAISE NOTICE '  CALL refresh_indicators(''1h'', ''BTC/USDT''); -- Specific symbol';
    RAISE NOTICE '=============================================================================';
END $$;
