# FlowGre Web API Reference

All endpoints are served by the built-in web server (Gorilla Mux router, `StrictSlash(true)`). The server binds to `0.0.0.0:8080` by default; override with `--web-ip` and `--web-port` flags or the equivalent YAML config keys.

**Machine-readable spec:** [openapi.yaml](openapi.yaml) (OpenAPI 3.0.3)

---

## Table of Contents

- [Authentication](#authentication)
- [Endpoints](#endpoints)
  - [`GET /`](#get-)
  - [`GET /health`](#get-health)
  - [`GET /stats`](#get-stats)
  - [`GET /stats/history`](#get-statshistory)
  - [`GET /dashboard`](#get-dashboard)
- [Data Types](#data-types)
- [Error Responses](#error-responses)

---

## Authentication

None. All endpoints are unauthenticated and intended for local monitoring. Bind the web server to `127.0.0.1` or a private interface if exposing to untrusted networks.

---

## Endpoints

### `GET /`

Root status endpoint. Returns a lightweight JSON health payload confirming the server is alive.

**Request:**

```
GET / HTTP/1.1
Host: localhost:8080
Accept: application/json
```

**Response:** `200 OK`

```json
{
  "status": "OK",
  "message": "Flowgre is flinging packets!"
}
```

**Content-Type:** `application/json`

---

### `GET /health`

Static health check. Identical structure to `/` but with a different message. Suitable for load-balancer probes and container orchestration health checks.

**Request:**

```
GET /health HTTP/1.1
Host: localhost:8080
Accept: application/json
```

**Response:** `200 OK`

```json
{
  "status": "OK",
  "message": "Everything is OK!"
}
```

**Content-Type:** `application/json`

---

### `GET /stats`

Returns current per-worker statistics and aggregate totals at the time of the request. The `workers` map is keyed by worker ID (JSON serialises integer keys as strings). Totals are recalculated from the live worker map on every stat update.

**Request:**

```
GET /stats HTTP/1.1
Host: localhost:8080
Accept: application/json
```

**Response:** `200 OK`

```json
{
  "workers": {
    "1": {
      "worker_id": 1,
      "source_id": 100,
      "flows_sent": 1500,
      "cycles": 100,
      "bytes_sent": 15728640
    },
    "2": {
      "worker_id": 2,
      "source_id": 200,
      "flows_sent": 1400,
      "cycles": 95,
      "bytes_sent": 14680064
    }
  },
  "totals": {
    "flows_sent": 2900,
    "cycles": 195,
    "bytes_sent": 30408704
  }
}
```

**Content-Type:** `application/json`

#### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `workers.<id>.worker_id` | int | Unique worker identifier |
| `workers.<id>.source_id` | int | Source flow identifier |
| `workers.<id>.flows_sent` | uint64 | Cumulative flows sent by this worker |
| `workers.<id>.cycles` | uint64 | Send cycles completed by this worker |
| `workers.<id>.bytes_sent` | uint64 | Total bytes sent by this worker |
| `totals.flows_sent` | uint64 | Sum of `flows_sent` across all workers |
| `totals.cycles` | uint64 | Sum of `cycles` across all workers |
| `totals.bytes_sent` | uint64 | Sum of `bytes_sent` across all workers |

---

### `GET /stats/history`

Returns a rolling buffer of historical stat snapshots. Each snapshot captures total counters and per-worker breakdown at a point in time. The buffer is capped at **300 entries** (~10 minutes at typical 2-second intervals). Oldest entries are evicted when the cap is reached.

**Request:**

```
GET /stats/history HTTP/1.1
Host: localhost:8080
Accept: application/json
```

**Response:** `200 OK`

```json
[
  {
    "timestamp": "2025-06-04T12:00:00Z",
    "totals": {
      "flows_sent": 100,
      "cycles": 10,
      "bytes_sent": 10240
    },
    "workers": {
      "1": {
        "worker_id": 1,
        "source_id": 100,
        "flows_sent": 100,
        "cycles": 10,
        "bytes_sent": 10240
      }
    }
  }
]
```

**Content-Type:** `application/json`

#### Notes

- Snapshots are appended each time a worker sends stats through the collector channel
- Timestamps are RFC 3339 formatted UTC
- The array is ordered oldest-first
- Empty array `[]` is returned before any stats have been collected

---

### `GET /dashboard`

Renders an HTML dashboard page with embedded Chart.js visualisations. Displays:

- Current worker stats table (worker ID, source ID, flows, cycles, bytes)
- Aggregate totals
- Configuration summary (protocol, server, ports, ranges, worker count)
- Barrage start time and human-readable uptime
- Live-updating charts (polls `/stats` via AJAX `setInterval`)
- Dark/light theme toggle

**Request:**

```
GET /dashboard HTTP/1.1
Host: localhost:8080
Accept: text/html
```

**Response:** `200 OK`

```html
<!DOCTYPE html>
<html lang="en">
<!-- Full dashboard HTML with embedded stats -->
</html>
```

**Content-Type:** `text/html`

#### Dashboard Polling

The dashboard page makes AJAX `GET /stats` requests at regular intervals (via JavaScript `setInterval`) to update charts in near-real-time. No WebSocket connection is used.

---

## Data Types

### Health

```json
{
  "status": "OK",
  "message": "string"
}
```

### WorkerStat

```json
{
  "worker_id": 1,
  "source_id": 100,
  "flows_sent": 1500,
  "cycles": 100,
  "bytes_sent": 15728640
}
```

### StatTotals

```json
{
  "flows_sent": 2900,
  "cycles": 195,
  "bytes_sent": 30408704
}
```

### StatSnapshot

```json
{
  "timestamp": "2025-06-04T12:00:00Z",
  "totals": { /* StatTotals */ },
  "workers": { /* map<integer, WorkerStat> */ }
}
```

---

## Error Responses

All endpoints may return `500 Internal Server Error` if JSON encoding fails during response construction. The response body is plain text:

```
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain

Internal server error
```

There are no client-side error codes (4xx) — the API does not accept request bodies or query parameters, so there is no user-input validation.

---

## Server Configuration

| Parameter | Default | CLI Flag | YAML Key | Description |
|-----------|---------|----------|----------|-------------|
| Bind IP | `0.0.0.0` | `--web-ip` | `web-ip` | Interface to listen on |
| Port | `8080` | `--web-port` | `web-port` | TCP port |
| Enabled | `false` | `--web` | `web` | Whether to start the web server |

### Timeout Settings

The HTTP server enforces strict timeouts on all connections:

| Setting | Value |
|---------|-------|
| Read Timeout | 5 seconds |
| Read Header Timeout | 5 seconds |
| Write Timeout | 5 seconds |
| Idle Timeout | 5 seconds |

These timeouts apply equally to all endpoints. Long-running requests (unlikely given the synchronous nature of the endpoints) will be terminated.
