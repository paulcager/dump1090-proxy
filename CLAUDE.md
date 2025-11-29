# dump1090-proxy

## Overview
A TCP proxy and aggregator for dump1090 BEAST protocol data. Connects to multiple remote dump1090 sources and redistributes the aggregated aircraft data to multiple downstream clients.

## Purpose
Enables centralized collection of ADS-B data from multiple distributed receivers:
- Aggregate data from multiple remote dump1090 instances
- Distribute combined data stream to multiple clients
- Monitor connection health and message throughput via Prometheus metrics
- Provide redundancy and failover capabilities

## Architecture

### Main Components

#### cmd/dump1090-proxy/main.go
Core proxy implementation with three main goroutine patterns:
1. **Listener goroutines**: Accept inbound client connections
2. **Remote goroutines**: Connect to upstream dump1090 sources
3. **Main goroutine**: Message distribution hub

#### beast/
BEAST protocol parser and message handling:
- Binary protocol used by dump1090 for efficient ADS-B data transmission
- Message framing and escaping logic
- Error recovery from malformed messages

#### sbs/
SBS-1 BaseStation protocol support (alternative text-based format)

### Data Flow
```
Remote dump1090 sources (BEAST TCP)
          ↓
    [runRemote goroutines]
          ↓
    newMessage channel
          ↓
    [Main distribution loop]
          ↓
    Connected clients
```

### Connection Management
- **Inbound connections**: Clients connect to listen address (default: `localhost:30005`)
  - Read is closed (write-only for clients)
  - TCP keepalive enabled (1 minute intervals)
  - 2-second write timeout per message
  - Failed writes remove client immediately

- **Outbound connections**: Proxy connects to remote sources
  - Automatic reconnection with exponential backoff (max 1 minute)
  - Hourly logging to reduce noise
  - TCP keepalive enabled
  - Write is closed (read-only from remotes)

### Metrics Exposed
- `messages_read`: Total dump1090 messages received from all sources
- `messages_written`: Total messages written to all clients
- `inbound_connections`: Current number of connected clients
- `outbound_connections`: Current number of active remote connections
- `ioerrors_total{op}`: IO errors by operation (read/write)

Metrics available at: `http://localhost:9798/metrics`

### Configuration
Key command-line flags:
- `--listen-address`: Local TCP address to listen on (default: `localhost:30005`)
- `--remote`: Remote server(s) to connect to (required, can be specified multiple times)
- `--dumpMessages`: Enable hex dump of all messages for debugging
- `--web.listen-address`: Prometheus metrics endpoint (default: `:9798`)
- `--web.telemetry-path`: Metrics path (default: `/metrics`)

Example:
```bash
dump1090_proxy \
  --listen-address=0.0.0.0:30005 \
  --remote=receiver1.local:30005 \
  --remote=receiver2.local:30005
```

## Dependencies
- `github.com/prometheus/*`: Prometheus client libraries
- `github.com/go-kit/log`: Structured logging
- `gopkg.in/alecthomas/kingpin.v2`: CLI flag parsing

## Docker Build
Multi-stage build:
1. **Build stage**: Uses `paulcager/go-base:latest`, compiles with CGO disabled
2. **Runtime stage**: Minimal `scratch` image with binary and CA certificates
3. **Exposed ports**:
   - `9798`: Prometheus metrics
   - `30005`: BEAST protocol data distribution
4. **Default configuration**: Connects to `pi-zero-flights.paulcager.org:30005` and `pi-zero-flights-2.paulcager.org:30005`

## Build & Deployment
Built using GitHub Actions workflow that creates multi-architecture images:
- Platforms: `linux/amd64`, `linux/arm64`
- Published to: `ghcr.io/<owner>/dump1090-proxy`
- Triggers: Push to main/master, PRs, manual workflow dispatch

## Use Cases
1. **Multi-site aggregation**: Combine data from receivers at different locations
2. **Redundancy**: Multiple upstream sources with automatic failover
3. **Fan-out**: Single aggregated stream to multiple flight tracking services
4. **Monitoring**: Prometheus metrics for receiver health and throughput

## Protocol Notes
### BEAST Protocol
- Binary format more efficient than SBS-1 text format
- Uses escape sequences (0x1A) for message framing
- Message types include position, velocity, identification, etc.
- Parser handles partial messages and synchronization errors

## Error Handling
- **Invalid messages**: Logged but skipped (except first message after connection)
- **Connection failures**: Exponential backoff retry with hourly logging
- **Write failures**: Client immediately disconnected
- **Read failures**: Remote connection closed and reconnected

## Development Notes
- Message channels are buffered (4 connections, 16 messages) to handle bursts
- Keepalive prevents idle connection timeouts through NAT/firewalls
- Write deadline prevents slow clients from blocking distribution
- First message validation prevents spurious warnings on connection
