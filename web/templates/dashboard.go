// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package templates

const DashboardTpl = `
<!DOCTYPE html>
<html lang="en">
<head>
<title>{{.Title}}</title>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Raleway">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6.5.1/css/all.min.css" integrity="sha384-t1nt8BQoYMLFN5p42tRAtuAAFQaCQODekUVeKKZrEnEyp4H2R0RHFz0KWpmj7i8g" crossorigin="anonymous">
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js" integrity="sha384-e6nUZLBkQ86NJ6TVVKAeSaK8jWa3NhkYWZFomE39AvDbQWeie9PlQqM3pmYW5d1g" crossorigin="anonymous"></script>
<style>
:root {
  --bg-primary: #1a1a2e;
  --bg-secondary: #16213e;
  --bg-card: #0f3460;
  --text-primary: #e4e4e4;
  --text-secondary: #a0a0a0;
  --accent-blue: #4fc3f7;
  --accent-green: #66bb6a;
  --accent-orange: #ffa726;
  --accent-red: #ef5350;
  --accent-purple: #ab47bc;
  --border-color: #2a2a4a;
  --shadow: 0 2px 8px rgba(0,0,0,0.3);
}

[data-theme="light"] {
  --bg-primary: #f5f5f5;
  --bg-secondary: #ffffff;
  --bg-card: #ffffff;
  --text-primary: #333333;
  --text-secondary: #666666;
  --border-color: #e0e0e0;
  --shadow: 0 2px 8px rgba(0,0,0,0.1);
}

* { margin: 0; padding: 0; box-sizing: border-box; }

body {
  font-family: "Raleway", -apple-system, BlinkMacSystemFont, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  min-height: 100vh;
}

/* Header */
.header {
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
  padding: 1rem 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: var(--shadow);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header h1 {
  font-size: 1.5rem;
  font-weight: 600;
}

.protocol-badge {
  background: var(--accent-blue);
  color: #000;
  padding: 0.25rem 0.75rem;
  border-radius: 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1.5rem;
}

.uptime {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.theme-toggle {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  padding: 0.5rem;
  border-radius: 0.5rem;
  cursor: pointer;
  transition: all 0.2s;
}

.theme-toggle:hover {
  background: var(--accent-blue);
  color: #000;
}

/* Main content */
.container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem;
}

/* Summary cards */
.cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}

.card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  padding: 1.5rem;
  box-shadow: var(--shadow);
  transition: transform 0.2s;
}

.card:hover {
  transform: translateY(-2px);
}

.card-icon {
  font-size: 1.5rem;
  margin-bottom: 0.5rem;
}

.card-value {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 0.25rem;
}

.card-label {
  color: var(--text-secondary);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.card-workers .card-icon { color: var(--accent-purple); }
.card-workers .card-value { color: var(--accent-purple); }
.card-flows .card-icon { color: var(--accent-blue); }
.card-flows .card-value { color: var(--accent-blue); }
.card-cycles .card-icon { color: var(--accent-green); }
.card-cycles .card-value { color: var(--accent-green); }
.card-bytes .card-icon { color: var(--accent-orange); }
.card-bytes .card-value { color: var(--accent-orange); }

/* Charts section */
.charts-section {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
  margin-bottom: 2rem;
}

.chart-container {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  padding: 1.5rem;
  box-shadow: var(--shadow);
}

.chart-container h3 {
  margin-bottom: 1rem;
  font-size: 1.1rem;
  color: var(--text-secondary);
}

.chart-wrapper {
  position: relative;
  height: 300px;
}

/* Worker table */
.table-section {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  padding: 1.5rem;
  box-shadow: var(--shadow);
  margin-bottom: 2rem;
  overflow-x: auto;
}

.table-section h3 {
  margin-bottom: 1rem;
  font-size: 1.1rem;
  color: var(--text-secondary);
}

table {
  width: 100%;
  border-collapse: collapse;
}

th, td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

th {
  color: var(--text-secondary);
  font-weight: 600;
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

tr:hover {
  background: rgba(255,255,255,0.02);
}

/* Config section */
.config-section {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 0.75rem;
  padding: 1.5rem;
  box-shadow: var(--shadow);
}

.config-section h3 {
  margin-bottom: 1rem;
  font-size: 1.1rem;
  color: var(--text-secondary);
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.config-label {
  color: var(--text-secondary);
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.config-value {
  font-size: 1rem;
  font-weight: 500;
}

/* Footer */
.footer {
  text-align: center;
  padding: 2rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
  border-top: 1px solid var(--border-color);
  margin-top: 2rem;
}

/* Responsive */
@media (max-width: 768px) {
  .header {
    flex-direction: column;
    gap: 1rem;
    text-align: center;
  }
  
  .header-right {
    justify-content: center;
  }
  
  .container {
    padding: 1rem;
  }
  
  .cards {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .chart-wrapper {
    height: 250px;
  }
}

@media (max-width: 480px) {
  .cards {
    grid-template-columns: 1fr;
  }
}
</style>
</head>
<body data-theme="dark">

<!-- Header -->
<div class="header">
  <div class="header-left">
    <h1><i class="fa-solid fa-gauge"></i> Flowgre Dashboard</h1>
    <span class="protocol-badge" id="protocolBadge">{{.Protocol}}</span>
  </div>
  <div class="header-right">
    <span class="uptime" id="uptimeDisplay">Uptime: {{.Uptime}}</span>
    <button class="theme-toggle" onclick="toggleTheme()" title="Toggle theme">
      <i class="fa-solid fa-sun" id="themeIcon"></i>
    </button>
  </div>
</div>

<!-- Main content -->
<div class="container">
  <!-- Summary cards -->
  <div class="cards">
    <div class="card card-workers">
      <div class="card-icon"><i class="fa-solid fa-users"></i></div>
      <div class="card-value" id="workersCount">{{.ConfigOut.Workers}}</div>
      <div class="card-label">Workers</div>
    </div>
    <div class="card card-flows">
      <div class="card-icon"><i class="fa-solid fa-share-nodes"></i></div>
      <div class="card-value" id="flowsCount">{{.StatsTotal.FlowsSent}}</div>
      <div class="card-label">Total Flows</div>
    </div>
    <div class="card card-cycles">
      <div class="card-icon"><i class="fa-solid fa-circle-nodes"></i></div>
      <div class="card-value" id="cyclesCount">{{.StatsTotal.Cycles}}</div>
      <div class="card-label">Cycles</div>
    </div>
    <div class="card card-bytes">
      <div class="card-icon"><i class="fa-solid fa-cloud-arrow-down"></i></div>
      <div class="card-value" id="bytesCount">{{formatBytes .StatsTotal.BytesSent}}</div>
      <div class="card-label">Bytes Sent</div>
    </div>
  </div>

  <!-- Charts -->
  <div class="charts-section">
    <div class="chart-container">
      <h3><i class="fa-solid fa-chart-line"></i> Flow Rate Over Time</h3>
      <div class="chart-wrapper">
        <canvas id="flowRateChart"></canvas>
      </div>
    </div>
  </div>

  <!-- Worker details -->
  <div class="table-section">
    <h3><i class="fa-solid fa-gears"></i> Worker Details</h3>
    <table>
      <thead>
        <tr>
          <th>Worker</th>
          <th>Source ID</th>
          <th>Flows Sent</th>
          <th>Cycles</th>
          <th>Bytes Sent</th>
          <th>Flows/sec</th>
        </tr>
      </thead>
      <tbody id="workerTable">
        {{ range $worker, $value := .StatsMapOut }}
        <tr>
          <td><i class="fa-solid fa-user" style="color: var(--accent-blue)"></i> #{{$worker}}</td>
          <td>{{$value.SourceID}}</td>
          <td>{{$value.FlowsSent}}</td>
          <td>{{$value.Cycles}}</td>
          <td>{{formatBytes $value.BytesSent}}</td>
          <td class="flows-per-sec">-</td>
        </tr>
        {{else}}
        <tr>
          <td colspan="6" style="text-align: center; color: var(--text-secondary);">No worker stats yet</td>
        </tr>
        {{end}}
      </tbody>
    </table>
  </div>

  <!-- Config details -->
  <div class="config-section">
    <h3><i class="fa-solid fa-wrench"></i> Configuration</h3>
    <div class="config-grid">
      <div class="config-item">
        <span class="config-label">Target Server</span>
        <span class="config-value">{{.ConfigOut.Server}}</span>
      </div>
      <div class="config-item">
        <span class="config-label">Target Port</span>
        <span class="config-value">{{.ConfigOut.DstPort}}</span>
      </div>
      <div class="config-item">
        <span class="config-label">Protocol</span>
        <span class="config-value">{{.Protocol}}</span>
      </div>
      <div class="config-item">
        <span class="config-label">Delay</span>
        <span class="config-value">{{.ConfigOut.Delay}} ms</span>
      </div>
      <div class="config-item">
        <span class="config-label">Workers</span>
        <span class="config-value">{{.ConfigOut.Workers}}</span>
      </div>
      <div class="config-item">
        <span class="config-label">Source Range</span>
        <span class="config-value">{{.ConfigOut.SrcRange}}</span>
      </div>
      <div class="config-item">
        <span class="config-label">Destination Range</span>
        <span class="config-value">{{.ConfigOut.DstRange}}</span>
      </div>
      <div class="config-item">
        <span class="config-label">Template Interval</span>
        <span class="config-value">{{.ConfigOut.TemplateInterval}}s</span>
      </div>
    </div>
  </div>
</div>

<!-- Footer -->
<div class="footer">
  <p>Flowgre Dashboard &mdash; Auto-refreshes every 2 seconds</p>
</div>

<script>
// Theme management
function toggleTheme() {
  const body = document.body;
  const icon = document.getElementById('themeIcon');
  if (body.dataset.theme === 'dark') {
    body.dataset.theme = 'light';
    icon.className = 'fa-solid fa-moon';
    updateChartTheme();
  } else {
    body.dataset.theme = 'dark';
    icon.className = 'fa-solid fa-sun';
    updateChartTheme();
  }
}

function updateChartTheme() {
  if (!flowRateChart) return;
  const isDark = document.body.dataset.theme === 'dark';
  const textColor = isDark ? '#a0a0a0' : '#666666';
  const gridColor = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)';
  
  flowRateChart.options.scales.x.ticks.color = textColor;
  flowRateChart.options.scales.y.ticks.color = textColor;
  flowRateChart.options.scales.x.grid.color = gridColor;
  flowRateChart.options.scales.y.grid.color = gridColor;
  flowRateChart.update('none');
}

// Format bytes
function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// Format number with commas
function formatNumber(num) {
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// Chart setup
let flowRateChart = null;
let previousTotals = null;
let chartData = {
  labels: [],
  flows: [],
  bytes: []
};

function initChart() {
  const ctx = document.getElementById('flowRateChart').getContext('2d');
  const isDark = document.body.dataset.theme === 'dark';
  const textColor = isDark ? '#a0a0a0' : '#666666';
  const gridColor = isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)';
  
  flowRateChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: chartData.labels,
      datasets: [
        {
          label: 'Flows/sec',
          data: chartData.flows,
          borderColor: '#4fc3f7',
          backgroundColor: 'rgba(79, 195, 247, 0.1)',
          tension: 0.3,
          fill: true
        },
        {
          label: 'MB/sec',
          data: chartData.bytes,
          borderColor: '#66bb6a',
          backgroundColor: 'rgba(102, 187, 106, 0.1)',
          tension: 0.3,
          fill: true
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 300 },
      interaction: {
        mode: 'index',
        intersect: false
      },
      plugins: {
        legend: {
          labels: { color: textColor }
        }
      },
      scales: {
        x: {
          ticks: { color: textColor, maxTicksLimit: 10 },
          grid: { color: gridColor }
        },
        y: {
          beginAtZero: true,
          ticks: { color: textColor },
          grid: { color: gridColor }
        }
      }
    }
  });
}

// Dashboard update
let lastUpdateTime = Date.now();

async function updateDashboard() {
  try {
    const response = await fetch('/stats');
    const data = await response.json();
    
    if (!data.totals || !data.workers) return;
    
    const totals = data.totals;
    const workers = data.workers;
    
    // Update summary cards
    document.getElementById('flowsCount').textContent = formatNumber(totals.flows_sent);
    document.getElementById('cyclesCount').textContent = formatNumber(totals.cycles);
    document.getElementById('bytesCount').textContent = formatBytes(totals.bytes_sent);
    
    // Calculate rates
    const now = Date.now();
    const timeDiff = (now - lastUpdateTime) / 1000; // seconds
    lastUpdateTime = now;
    
    let flowsPerSec = 0;
    let bytesPerSec = 0;
    
    if (previousTotals && timeDiff > 0) {
      flowsPerSec = Math.round((totals.flows_sent - previousTotals.flows_sent) / timeDiff);
      bytesPerSec = (totals.bytes_sent - previousTotals.bytes_sent) / timeDiff;
    }
    
    previousTotals = { ...totals };
    
    // Update chart data
    const timeLabel = new Date().toLocaleTimeString();
    chartData.labels.push(timeLabel);
    chartData.flows.push(flowsPerSec);
    chartData.bytes.push(bytesPerSec / (1024 * 1024)); // Convert to MB
    
    // Keep only last 60 data points (2 minutes at 2-second intervals)
    if (chartData.labels.length > 60) {
      chartData.labels.shift();
      chartData.flows.shift();
      chartData.bytes.shift();
    }
    
    // Update chart
    if (flowRateChart) {
      flowRateChart.data.labels = chartData.labels;
      flowRateChart.data.datasets[0].data = chartData.flows;
      flowRateChart.data.datasets[1].data = chartData.bytes;
      flowRateChart.update('none');
    }
    
    // Update worker table
    updateWorkerTable(workers, timeDiff);
    
  } catch (error) {
    console.error('Failed to update dashboard:', error);
  }
}

function updateWorkerTable(workers, timeDiff) {
  const tbody = document.getElementById('workerTable');
  const workerIds = Object.keys(workers).sort((a, b) => parseInt(a) - parseInt(b));
  
  if (workerIds.length === 0) {
    tbody.innerHTML = '<tr><td colspan="6" style="text-align: center; color: var(--text-secondary);">No worker stats yet</td></tr>';
    return;
  }
  
  let html = '';
  workerIds.forEach(id => {
    const w = workers[id];
    let flowsPerSec = '-';
    if (timeDiff > 0 && w.flows_sent > 0) {
      // This is a rough estimate since we don't track per-worker previous totals
      flowsPerSec = Math.round(w.flows_sent / Math.max(1, timeDiff * 100)).toString();
    }
    
    html += '<tr>';
    html += '<td><i class="fa-solid fa-user" style="color: var(--accent-blue)"></i> #' + id + '</td>';
    html += '<td>' + w.source_id + '</td>';
    html += '<td>' + formatNumber(w.flows_sent) + '</td>';
    html += '<td>' + formatNumber(w.cycles) + '</td>';
    html += '<td>' + formatBytes(w.bytes_sent) + '</td>';
    html += '<td>' + flowsPerSec + '</td>';
    html += '</tr>';
  });
  
  tbody.innerHTML = html;
}

// Initialize
document.addEventListener('DOMContentLoaded', function() {
  initChart();
  
  // Initial data fetch
  updateDashboard();
  
  // Poll every 2 seconds
  setInterval(updateDashboard, 2000);
});
</script>
</body>
</html>
`
