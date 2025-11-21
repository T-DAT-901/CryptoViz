# CryptoViz - Docker Desktop Troubleshooting Guide

> **Quick Start**:
> - **Mac**: Run `make mac-start` instead of `make start`
> - **Windows/WSL2**: Run `make windows-start` instead of `make start`

## Table of Contents
- [Docker Desktop Issues (Mac + Windows/WSL2)](#docker-desktop-issues-mac--windowswsl2)
- [Quick Fixes](#quick-fixes)
- [Debugging Commands](#debugging-commands)
- [Understanding Backfill Behavior](#understanding-backfill-behavior)
- [Common Error Messages](#common-error-messages)
- [Platform-Specific Notes](#platform-specific-notes)

---

## Docker Desktop Issues (Mac + Windows/WSL2)

### Issue 1: Network Not Found Error
```
Error response from daemon: network b4f2c0f8e93c not found
```

**Cause**: Stale Docker network references from previous runs

**Fix**:
```bash
# Clean stale networks
docker network prune -f

# Then start with platform-specific commands
# Mac:
make mac-start

# Windows/WSL2:
make windows-start
```

### Issue 2: Mount Propagation Error
```
Error response from daemon: path / is mounted on / but it is not a shared or slave mount
```

**Cause**: Docker Desktop (Mac/Windows WSL2) doesn't support `rslave` bind propagation used by `node-exporter`

**Fix**: The `docker-compose.mac.yml` override file automatically fixes this. Use:
```bash
# Mac:
make mac-start              # Start application
make mac-start-monitoring   # Start monitoring stack

# Windows/WSL2:
make windows-start              # Start application
make windows-start-monitoring   # Start monitoring stack
```

---

## Quick Fixes

### Complete Fresh Start (Recommended for Docker Desktop)

```bash
# Mac:
# 1. Clean everything
make mac-clean

# 2. Clean stale networks (important!)
docker network prune -f

# 3. Rebuild and start
make mac-start

# 4. Start monitoring (optional)
make mac-start-monitoring
```

```bash
# Windows/WSL2:
# 1. Clean everything
make windows-clean

# 2. Clean stale networks (important!)
docker network prune -f

# 3. Rebuild and start
make windows-start

# 4. Start monitoring (optional)
make windows-start-monitoring
```

### Reset Historical Data Collection

If you're not getting historical data:

```bash
# Mac:
# 1. Remove backfill state file
make mac-reset-backfill

# 2. Check if backfill is running
make debug-backfill

# 3. Monitor logs in real-time
docker logs -f cryptoviz-data-collector | grep -i backfill
```

```bash
# Windows/WSL2:
# 1. Remove backfill state file
make windows-reset-backfill

# 2. Check if backfill is running
make debug-backfill

# 3. Monitor logs in real-time
docker logs -f cryptoviz-data-collector | grep -i backfill
```

---

## Debugging Commands

### Check Backfill Status
```bash
make debug-backfill
```
This shows:
- Whether `backfill_state.json` exists
- ENABLE_BACKFILL setting in `.env`
- Recent backfill logs
- Data-collector container status

### Verify Historical Data in Database
```bash
make debug-timescale
```
This shows:
- Count of historical candles per interval
- Top symbols collected
- Recent data (last hour)

### Clean Docker Resources
```bash
make check-docker-resources
```
This helps clean:
- Stale networks (automatic)
- Orphaned volumes (manual confirmation)

### Monitor Logs in Real-Time
```bash
# All services
make logs

# Specific service
make logs-service SERVICE=data-collector

# Just backfill messages
docker logs -f cryptoviz-data-collector 2>&1 | grep -i backfill
```

---

## Understanding Backfill Behavior

### What is Backfill?

Backfill is the process of fetching historical cryptocurrency data (candles) when the application starts for the first time or when the state file is missing.

### How It Works

1. **On First Start**:
   - Data-collector checks for `services/data-collector/backfill_state.json`
   - If missing or incomplete, starts fetching historical data
   - Fetches data for all configured timeframes (1m, 5m, 15m, 1h, 1d)
   - Default lookback: 40 days (configurable via `BACKFILL_LOOKBACK_DAYS`)

2. **Progress Tracking**:
   - State file tracks last timestamp for each symbol/timeframe
   - Saves progress after each batch (1000 candles)
   - Allows resuming if interrupted

3. **On Subsequent Starts**:
   - Reads state file
   - Checks if backfill is complete (within 1 hour of target)
   - Skips backfill if already complete
   - Only collects real-time data

### Expected Behavior

**Normal Behavior** (You SHOULD get historical data):
```bash
# Check state file exists
ls -la services/data-collector/backfill_state.json

# Should show file with timestamps
cat services/data-collector/backfill_state.json | head -20

# Verify data in database
make debug-timescale
# Should show thousands of candles
```

**Abnormal Behavior** (Missing historical data):
```bash
# State file missing or empty
ls -la services/data-collector/backfill_state.json
# File not found or size 0

# No historical data in database
make debug-timescale
# Shows 0 or very few candles

# Logs show backfill skipped or failed
docker logs cryptoviz-data-collector 2>&1 | grep -i backfill
```

### Configuration

Check your `.env` file:
```bash
# Backfill must be enabled
ENABLE_BACKFILL=true

# Lookback period (days)
BACKFILL_LOOKBACK_DAYS=40

# Timeframes to fetch
BACKFILL_TIMEFRAMES=1m,5m,15m,1h,1d
```

### Forcing a Re-run

To force backfill to run again:
```bash
# Method 1: Use Makefile command
make mac-reset-backfill

# Method 2: Manual removal
rm -f services/data-collector/backfill_state.json
docker-compose restart data-collector

# Method 3: Complete clean start
make mac-clean
make mac-start
```

---

## Common Error Messages

### "KeyError: 'ContainerConfig'"
**Cause**: Corrupted Docker image cache after Dockerfile changes

**Fix**:
```bash
make mac-clean
make mac-start
```

### "Backend non disponible" when testing API
**Cause**: Backend-go not started or not healthy

**Fix**:
```bash
# Check status
docker-compose ps

# Check logs
docker logs cryptoviz-backend-go

# Restart if needed
docker-compose restart backend-go
```

### No historical values on frontend
**Possible Causes**:
1. Backfill didn't run → Check `make debug-backfill`
2. Backend not querying DB correctly → Check `make debug-timescale`
3. Frontend not displaying data → Check browser console

**Debugging Steps**:
```bash
# 1. Verify backfill ran
make debug-backfill

# 2. Verify data exists
make debug-timescale

# 3. Test backend endpoint directly
curl "http://localhost:8080/api/v1/crypto/candles?symbol=BTC/USDT&interval=1m&limit=100" | jq

# 4. Check frontend logs
docker logs cryptoviz-frontend-vue

# 5. Check browser console (F12) for errors
```

---

## Platform-Specific Notes

### Mac Docker Desktop

**Architecture**: Uses HyperKit or Virtualization.framework for VM
**Filesystem**: osxfs or virtio-fs (newer versions)
**Limitations**:
- No rslave mount propagation support
- No direct access to /dev/kmsg
- Slightly slower I/O due to filesystem virtualization

**Commands**:
- `make mac-start`
- `make mac-start-monitoring`
- `make mac-clean`
- `make mac-reset-backfill`

### Windows Docker Desktop (WSL2)

**Architecture**: Uses WSL2 (Windows Subsystem for Linux) backend
**Filesystem**: virtio-fs (fast, native-like performance)
**Limitations**:
- Same mount propagation restrictions as Mac
- No direct device access
- WSL2-specific networking considerations

**Detection**: System shows as "Linux" but kernel includes "microsoft"
**Commands**:
- `make windows-start`
- `make windows-start-monitoring`
- `make windows-clean`
- `make windows-reset-backfill`

### Docker Desktop Limitations (Both Platforms)

Due to Docker Desktop architecture (virtualization), some monitoring features are limited:

#### node-exporter
- **Limitation**: Reduced system metrics (no rslave mount access)
- **Impact**: Fewer disk and filesystem metrics
- **Workaround**: None needed, core functionality unaffected

#### cAdvisor
- **Limitation**: Non-privileged mode, no /dev/kmsg access
- **Impact**: Fewer low-level container metrics
- **Workaround**: None needed, basic container metrics still available

#### Performance
- **Limitation**: Docker Desktop uses virtualization (slower than native Linux)
- **Impact**: Slightly slower build times and I/O operations
- **Workaround**: Use `.dockerignore` and multi-stage builds (already implemented)

---

## Getting Help

### 1. Check System Info
```bash
make info
```

### 2. View All Available Commands
```bash
make help
```

### 3. Common Diagnostic Sequence
```bash
# Full diagnostic run
make debug-backfill
make debug-timescale
make check-docker-resources
docker-compose ps
```

### 4. Collect Logs for Debugging
```bash
# Save all logs to file
docker-compose logs > logs_$(date +%Y%m%d_%H%M%S).txt

# Save specific service logs
docker logs cryptoviz-data-collector > data-collector_$(date +%Y%m%d_%H%M%S).log
```

---

## Summary of Commands

### Mac Docker Desktop

| Command | Purpose |
|---------|---------|
| `make mac-start` | Start with Mac overrides (recommended) |
| `make mac-start-monitoring` | Start monitoring stack on Mac |
| `make mac-build` | Build images with Mac overrides |
| `make mac-clean` | Clean Docker + stale networks |
| `make mac-reset-backfill` | Reset backfill state and restart |
| `make debug-backfill` | Check backfill status |
| `make debug-timescale` | Verify historical data |
| `make check-docker-resources` | Clean stale Docker resources |

### Windows Docker Desktop (WSL2)

| Command | Purpose |
|---------|---------|
| `make windows-start` | Start with Windows/WSL2 overrides (recommended) |
| `make windows-start-monitoring` | Start monitoring stack on Windows/WSL2 |
| `make windows-build` | Build images with Windows/WSL2 overrides |
| `make windows-clean` | Clean Docker + stale networks |
| `make windows-reset-backfill` | Reset backfill state and restart |
| `make debug-backfill` | Check backfill status (all platforms) |
| `make debug-timescale` | Verify historical data (all platforms) |
| `make check-docker-resources` | Clean stale Docker resources (all platforms) |

### Platform Detection

Run `make info` or `make help` to see platform-specific commands based on auto-detection.

---

**Last Updated**: 2025-11-21
**For**: Docker Desktop users (Mac + Windows/WSL2)
**CryptoViz Version**: 0.7.0
