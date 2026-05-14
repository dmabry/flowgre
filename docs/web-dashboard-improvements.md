# Web Dashboard Improvements

**Status:** Proposed  
**Priority:** Medium  
**Effort:** Half-day to full day  
**Author:** Sparky  
**Created:** 2026-05-14

---

## Problem Statement

The barrage web dashboard is functional but barebones. It's a static HTML page with a 30-second auto-refresh that shows:

1. **4 summary cards** — Workers, Flows, Cycles, BytesSent
2. **Worker details table** — per-worker stats
3. **Config details table** — target server, port, delay, workers

The dashboard has several limitations:

1. **No real-time visualization** — stats are static numbers that update every 30 seconds via full page reload. No charts, no flow rate over time, no trends.
2. **No historical data** — the dashboard shows current totals but has no memory of past performance. You can't see "how fast were we 5 minutes ago?"
3. **No per-worker granularity in visualization** — the worker table is a static list with no visual comparison between workers.
4. **No health monitoring** — the health endpoint exists (`/health`) but isn't surfaced in the dashboard.
5. **No protocol indicator** — the dashboard doesn't show whether the barrage is running NetFlow v9 or IPFIX.

---

## Current Architecture

### Backend

| File | Role | Key Endpoints |
|------|------|---------------|
| `web/web.go` | HTTP server, route setup | `/`, `/health`, `/stats`, `/dashboard` |
| `stats/collector.go` | Stats aggregation, HTTP handlers | `StatsHandler()`, `DashboardHandler()` |
| `models/models.go` | Data structures | `WorkerStat`, `StatTotals`, `DashboardPage`, `Health` |

### Data Flow

```
Worker → StatsChan → Collector.Run() → StatsMap + StatsTotals
                                              ↓
                                    StatsHandler (JSON)     DashboardHandler (HTML)
                                              ↓                    ↓
                                    GET /stats              GET /dashboard
```

### Current Dashboard Template

`web/templates/dashboard.go` — a single Go template string (201 lines) that:
- Uses w3.css for styling
- Auto-refreshes every 30 seconds via jQuery
- Renders 4 summary cards + worker table + config table
- No charts, no real-time updates, no history

### Current Stats Data

```go
type WorkerStat struct {
    WorkerID  int    `json:"worker_id,omitempty"`
    SourceID  int    `json:"source_id,omitempty"`
    FlowsSent uint64 `json:"flows_sent,omitempty"`
    Cycles    uint64 `json:"cycles,omitempty"`
    BytesSent uint64 `json:"bytes_sent,omitempty"`
}

type StatTotals struct {
    FlowsSent uint64
    Cycles    uint64
    BytesSent uint64
}
```

### Current API Endpoints

| Endpoint | Method | Response | Description |
|----------|--------|----------|-------------|
| `/` | GET | JSON `Health` | Static health check |
| `/health` | GET | JSON `Health` | Static health check |
| `/stats` | GET | JSON `map[int]WorkerStat` | Per-worker stats |
| `/dashboard` | GET | HTML | Full dashboard page |

---

## Proposed Improvements

### Tier 1: Real-Time Stats API (High Impact, Low Effort)

Add a `/stats/history` endpoint that returns time-series data for charting.

#### Changes

**File: `stats/collector.go`** (modify)

Add a history buffer to the `Collector`:

```go
type StatSnapshot struct {
    Timestamp time.Time     `json:"timestamp"`
    Totals    models.StatTotals
    Workers   map[int]models.WorkerStat
}

type Collector struct {
    mu          sync.RWMutex
    StatsMap    map[int]models.WorkerStat
    StatsChan   chan models.WorkerStat
    StatsTotals models.StatTotals
    Config      *models.Config
    History     []StatSnapshot        // rolling history
    MaxHistory  int                    // max snapshots to keep
}
```

Append to history on each stats update in `Run()`:

```go
// In Collector.Run(), after updating StatsMap and StatsTotals:
sc.mu.Lock()
snapshot := StatSnapshot{
    Timestamp: time.Now(),
    Totals:    sc.StatsTotals,
    Workers:   copyMap(sc.StatsMap), // deep copy
}
sc.History = append(sc.History, snapshot)
if len(sc.History) > sc.MaxHistory {
    sc.History = sc.History[len(sc.History)-sc.MaxHistory:]
}
sc.mu.Unlock()
```

**File: `web/web.go`** (modify)

Add `/stats/history` endpoint:

```go
router.HandleFunc("/stats/history", sc.HistoryHandler).Methods("GET")
```

**File: `stats/collector.go`** (add)

```go
// HistoryHandler returns time-series stats for charting.
func (sc *Collector) HistoryHandler(w http.ResponseWriter, r *http.Request) {
    sc.mu.RLock()
    defer sc.mu.RUnlock()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(sc.History)
}
```

**Tests:**
- `TestCollector_History_Appends` — verify snapshots are appended
- `TestCollector_History_Rotates` — verify old snapshots are dropped when MaxHistory is exceeded
- `TestHistoryHandler` — verify JSON response format

### Tier 2: Real-Time Dashboard with Auto-Polling (Medium Impact, Medium Effort)

Replace the 30-second full-page reload with AJAX-based polling that updates the dashboard in-place.

#### Changes

**File: `web/templates/dashboard.go`** (modify)

Replace the jQuery `location.reload()` with `setInterval` that fetches `/stats` and `/stats/history` via `fetch()`:

```javascript
// Replace the 30-second reload with 2-second AJAX polling
var pollInterval = setInterval(function() {
    fetch('/stats')
        .then(r => r.json())
        .then(data => updateDashboard(data));
}, 2000);

function updateDashboard(stats) {
    // Update summary cards
    // Update worker table
    // Update charts (Tier 3)
}
```

**Benefits:**
- Dashboard updates every 2 seconds instead of 30
- No full page reload — smoother UX
- Lower server load (JSON vs full HTML render)

**Tests:**
- Manual visual verification (UI changes are hard to unit test)
- `TestStatsHandler_JSON_Format` — verify JSON structure matches frontend expectations

### Tier 3: Real-Time Flow Rate Charts (High Impact, Medium Effort)

Add a lightweight charting library (Chart.js via CDN) to visualize flow rate over time.

#### Changes

**File: `web/templates/dashboard.go`** (modify)

Add Chart.js and a canvas element:

```html
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
<canvas id="flowRateChart"></canvas>
```

Initialize chart with history data:

```javascript
var flowRateChart = new Chart(document.getElementById('flowRateChart'), {
    type: 'line',
    data: {
        labels: [],
        datasets: [{
            label: 'Flows/sec',
            data: [],
            borderColor: '#2196F3',
            tension: 0.1
        }, {
            label: 'MB/sec',
            data: [],
            borderColor: '#4CAF50',
            tension: 0.1
        }]
    },
    options: {
        responsive: true,
        animation: false, // disable for performance with frequent updates
        scales: {
            y: { beginAtZero: true }
        }
    }
});
```

Update chart on each poll:

```javascript
function updateChart(history) {
    var labels = history.map(s => new Date(s.timestamp).toLocaleTimeString());
    var flows = history.map(s => s.totals.flowsSent);
    var bytes = history.map(s => s.totals.bytesSent / 1024 / 1024);
    
    flowRateChart.data.labels = labels;
    flowRateChart.data.datasets[0].data = flows;
    flowRateChart.data.datasets[1].data = bytes;
    flowRateChart.update();
}
```

**Tests:**
- Manual visual verification
- `TestHistoryHandler_Data_Format` — verify history data is chart-compatible

### Tier 4: Dashboard Polish (Low Impact, Low Effort)

Small UX improvements to the dashboard:

1. **Protocol indicator** — show "NetFlow v9" or "IPFIX" in the header
2. **Uptime display** — show how long the barrage has been running
3. **Per-worker flow rate** — calculate flows/sec per worker
4. **Dark theme toggle** — flowgre users work in dark terminals; the dashboard should match
5. **Responsive layout** — the current w3.css layout is already responsive, but the table could be better on mobile

#### Changes

**File: `models/models.go`** (modify)

Add to `DashboardPage`:

```go
type DashboardPage struct {
    Title       string
    Comment     string
    HealthOut   Health
    ConfigOut   *Config
    StatsMapOut map[int]WorkerStat
    StatsTotal  StatTotals
    Protocol    string    `json:"protocol"`    // "netflow" or "ipfix"
    StartTime   time.Time `json:"start_time"`  // when barrage started
    Uptime      string    `json:"uptime"`      // human-readable uptime
}
```

**File: `stats/collector.go`** (modify)

Populate new fields in `DashboardHandler`:

```go
d.Protocol = sc.Config.Protocol // add to Config struct
d.StartTime = sc.StartTime      // set at Collector creation
d.Uptime = time.Since(sc.StartTime).String()
```

**File: `models/models.go`** (modify)

Add to `Config`:

```go
type Config struct {
    // ... existing fields
    Protocol string `json:"protocol,omitempty"` // "netflow" or "ipfix"
}
```

**File: `web/templates/dashboard.go`** (modify)

Add protocol badge and uptime display to header.

**Tests:**
- `TestDashboardPage_NewFields` — verify new fields are populated

---

## Implementation Plan

### Phase 1: Stats History API (Tier 1) — 1 hour

1. Add `StatSnapshot` struct to `stats/collector.go`
2. Add `History` and `MaxHistory` to `Collector` struct
3. Append snapshots in `Run()` after each stats update
4. Add `/stats/history` endpoint to `web/web.go`
5. Add `HistoryHandler()` to `stats/collector.go`
6. Write tests

### Phase 2: Real-Time Polling (Tier 2) — 1 hour

1. Replace 30-second `location.reload()` with 2-second `fetch()` polling
2. Update dashboard elements via DOM manipulation
3. Remove jQuery dependency (only used for the reload timer)
4. Write tests for JSON format compatibility

### Phase 3: Charts (Tier 3) — 1 hour

1. Add Chart.js CDN to dashboard template
2. Add canvas element for flow rate chart
3. Initialize chart with history data
4. Update chart on each poll cycle
5. Manual visual verification

### Phase 4: Polish (Tier 4) — 30 minutes

1. Add protocol indicator to header
2. Add uptime display
3. Add dark theme CSS (minimal — just invert colors)
4. Update `Config` and `DashboardPage` structs
5. Write tests

---

## File Changes Summary

| File | Action | Tiers | Description |
|------|--------|-------|-------------|
| `stats/collector.go` | **Modify** | 1, 2, 4 | Add history buffer, `HistoryHandler()`, populate new fields |
| `web/web.go` | **Modify** | 1, 2 | Add `/stats/history` endpoint |
| `web/templates/dashboard.go` | **Modify** | 2, 3, 4 | Replace reload with polling, add charts, add polish |
| `models/models.go` | **Modify** | 4 | Add `Protocol`, `StartTime`, `Uptime` to `DashboardPage` and `Config` |
| `barrage/barrage.go` | **Modify** | 4 | Pass protocol to `Collector` |
| `stats/collector_test.go` | **Modify** | 1 | Add history tests |
| `web/web_test.go` | **Modify** | 2 | Add JSON format tests |

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Chart.js CDN unavailable | Dashboard broken | Inline minimal chart code or use canvas directly as fallback |
| History buffer memory leak | High memory usage under long runs | `MaxHistory` caps the buffer (e.g., 300 snapshots = 10 minutes at 2s intervals) |
| Dashboard template gets large | Hard to maintain | Extract chart JS to separate file, keep template focused on HTML structure |
| Race conditions on history | Data corruption | Protected by existing `sync.RWMutex` on `Collector` |

---

## Acceptance Criteria

- [ ] `/stats/history` returns valid JSON array of `StatSnapshot` objects
- [ ] Dashboard auto-updates every 2 seconds without full page reload
- [ ] Flow rate chart shows flows/sec and MB/sec over time
- [ ] Protocol indicator shows "NetFlow v9" or "IPFIX"
- [ ] Uptime display shows elapsed time since barrage start
- [ ] All existing tests pass with `-race` detector
- [ ] Dashboard works on mobile browsers (responsive layout)
- [ ] No external dependencies beyond Chart.js CDN (optional)

---

## Future Enhancements (Out of Scope)

- **WebSocket push** — server pushes updates instead of client polling
- **Per-worker charts** — individual flow rate charts per worker
- **Alert thresholds** — configurable alerts for flow rate drops
- **Export to CSV** — download historical stats
- **Multiple barrage comparison** — compare two barrages side-by-side
- **Grafana integration** — expose stats as Prometheus metrics
