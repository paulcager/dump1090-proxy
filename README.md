# dump1090-proxy

A TCP proxy and aggregator for dump1090 BEAST protocol data. Collects ADS-B aircraft data from multiple remote dump1090 sources and redistributes the combined stream to multiple clients.

## Features

- Connects to multiple remote dump1090 BEAST sources
- Aggregates and redistributes data to multiple clients
- Automatic reconnection with exponential backoff
- Prometheus metrics for monitoring
- Multi-architecture Docker images (AMD64, ARM64)
- Minimal resource footprint

## Usage

```bash
dump1090_proxy \
  --listen-address=0.0.0.0:30005 \
  --remote=receiver1.example.com:30005 \
  --remote=receiver2.example.com:30005
```

### Command-line Flags

```
  --listen-address=ADDR           Local address to listen on (default: localhost:30005)
  --remote=HOST:PORT              Remote dump1090 server (required, can be specified multiple times)
  --web.listen-address=ADDR       Prometheus metrics endpoint (default: :9798)
  --web.telemetry-path=PATH       Metrics path (default: /metrics)
  --dumpMessages                  Enable hex dump of all messages for debugging
  -h, --help                      Show help
```

## Docker

Multi-architecture images are available via GitHub Container Registry:

```bash
docker pull ghcr.io/<username>/dump1090-proxy:latest
```

Run the proxy:
```bash
docker run -d \
  -p 30005:30005 \
  -p 9798:9798 \
  ghcr.io/<username>/dump1090-proxy:latest \
    --listen-address=0.0.0.0:30005 \
    --remote=receiver1.local:30005 \
    --remote=receiver2.local:30005
```

## Metrics

Prometheus metrics exposed at `:9798/metrics`:

- `messages_read` - Total messages received from all remote sources
- `messages_written` - Total messages written to all clients
- `inbound_connections` - Current number of connected clients
- `outbound_connections` - Current number of active remote connections
- `ioerrors_total{op}` - IO errors by operation type

## Architecture

The proxy operates with three main components:

1. **Listeners**: Accept client connections on the listen address
2. **Remote connectors**: Connect to upstream dump1090 sources
3. **Message distributor**: Central hub that receives messages and distributes to all clients

All remote sources are aggregated into a single stream distributed to all connected clients.

## Use Cases

- **Multi-site aggregation**: Combine data from geographically distributed receivers
- **Redundancy**: Multiple upstream sources with automatic failover
- **Load distribution**: Single source to multiple flight tracking services
- **Monitoring**: Track receiver health and message throughput

## Protocol

Uses the BEAST binary protocol for efficient ADS-B data transmission. The BEAST protocol is more efficient than text-based SBS-1 format and is the standard output format for dump1090.

## Building

```bash
go build -v -o dump1090_proxy ./cmd/dump1090-proxy
```

## Service Documentation

See [SERVICES.md](SERVICES.md) for documentation about FlightRadar24, RadarBox, and PiAware service integration.
