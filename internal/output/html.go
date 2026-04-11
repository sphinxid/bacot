package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/sphinxid/bacot/internal/metrics"
	"github.com/sphinxid/bacot/internal/thresholds"
)

// WriteHTML generates a self-contained HTML report and writes it to path.
func WriteHTML(path string, testName string, collector *metrics.Collector, thresholdResults []thresholds.Result, elapsed time.Duration, startedAt time.Time) error {
	report := buildJSONReport(testName, collector, thresholdResults, elapsed, startedAt)

	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshaling report data: %w", err)
	}

	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("parsing HTML template: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating HTML report file %s: %w", path, err)
	}
	defer f.Close()

	data := struct {
		Title      string
		ReportJSON string
		GeneratedAt string
	}{
		Title:      testName,
		ReportJSON: string(reportJSON),
		GeneratedAt: startedAt.Format(time.RFC1123),
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("rendering HTML template: %w", err)
	}
	return nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>bacot — {{.Title}}</title>
<script>
// Chart.js 4.x embedded (minified)
// Using CDN fallback with offline note
</script>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f1117; color: #e2e8f0; min-height: 100vh; }
  .header { background: linear-gradient(135deg, #1a1f2e 0%, #16213e 100%); padding: 24px 32px; border-bottom: 1px solid #2d3748; }
  .header h1 { font-size: 1.5rem; color: #63b3ed; font-weight: 700; }
  .header p { color: #718096; font-size: 0.875rem; margin-top: 4px; }
  .container { max-width: 1200px; margin: 0 auto; padding: 24px 32px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
  .card { background: #1a1f2e; border: 1px solid #2d3748; border-radius: 8px; padding: 20px; }
  .card h3 { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em; color: #718096; margin-bottom: 8px; }
  .card .value { font-size: 2rem; font-weight: 700; color: #63b3ed; }
  .card .sub { font-size: 0.875rem; color: #718096; margin-top: 4px; }
  .section { background: #1a1f2e; border: 1px solid #2d3748; border-radius: 8px; padding: 24px; margin-bottom: 24px; }
  .section h2 { font-size: 1rem; font-weight: 600; color: #e2e8f0; margin-bottom: 16px; border-bottom: 1px solid #2d3748; padding-bottom: 12px; }
  .chart-container { position: relative; height: 250px; }
  table { width: 100%; border-collapse: collapse; font-size: 0.875rem; }
  th { text-align: left; padding: 8px 12px; color: #718096; font-weight: 500; border-bottom: 1px solid #2d3748; }
  td { padding: 8px 12px; border-bottom: 1px solid #1e2535; }
  tr:last-child td { border-bottom: none; }
  .badge { display: inline-flex; align-items: center; gap: 4px; padding: 2px 8px; border-radius: 4px; font-size: 0.75rem; font-weight: 600; }
  .badge-pass { background: #1c3a2f; color: #48bb78; }
  .badge-fail { background: #3a1c1c; color: #fc8181; }
  .badge-2xx { background: #1c3a2f; color: #48bb78; }
  .badge-4xx { background: #3a2e1c; color: #f6ad55; }
  .badge-5xx { background: #3a1c1c; color: #fc8181; }
  .charts-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
  @media (max-width: 768px) { .charts-row { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<div class="header">
  <h1>&#9889; bacot — Load Test Report</h1>
  <p>{{.Title}} &middot; Generated {{.GeneratedAt}}</p>
</div>
<div class="container">
  <div id="summary-cards" class="grid"></div>
  <div class="charts-row">
    <div class="section"><h2>Requests per Second</h2><div class="chart-container"><canvas id="rpsChart"></canvas></div></div>
    <div class="section"><h2>Latency Over Time (p95)</h2><div class="chart-container"><canvas id="latencyChart"></canvas></div></div>
  </div>
  <div class="charts-row">
    <div class="section"><h2>Error Rate Over Time (%)</h2><div class="chart-container"><canvas id="errorChart"></canvas></div></div>
    <div class="section"><h2>Latency Distribution</h2><div class="chart-container"><canvas id="latencyDistChart"></canvas></div></div>
  </div>
  <div class="section"><h2>Thresholds</h2><div id="thresholds-table"></div></div>
  <div class="section"><h2>Scenarios</h2><div id="scenarios-table"></div></div>
  <div class="section"><h2>Status Codes</h2><div id="status-table"></div></div>
</div>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
<script>
const REPORT = {{.ReportJSON}};

// Summary cards
const summary = REPORT.summary;
const cards = [
  { label: "Total Requests", value: summary.total_requests.toLocaleString(), sub: summary.rps.toFixed(1) + "/s avg" },
  { label: "Failure Rate", value: summary.failure_rate_pct.toFixed(2) + "%", sub: summary.total_failures.toLocaleString() + " failures" },
  { label: "Avg Latency", value: summary.latency.avg_ms.toFixed(0) + "ms", sub: "p95: " + summary.latency.p95_ms.toFixed(0) + "ms" },
  { label: "p99 Latency", value: summary.latency.p99_ms.toFixed(0) + "ms", sub: "min: " + summary.latency.min_ms.toFixed(0) + "ms" },
  { label: "Data Sent", value: fmtBytes(summary.bytes_sent), sub: "↑ transferred" },
  { label: "Data Received", value: fmtBytes(summary.bytes_received), sub: "↓ received" },
];
const cardsEl = document.getElementById("summary-cards");
cards.forEach(c => {
  const d = document.createElement("div");
  d.className = "card";
  d.innerHTML = '<h3>' + c.label + '</h3><div class="value">' + c.value + '</div><div class="sub">' + c.sub + '</div>';
  cardsEl.appendChild(d);
});

// Charts
const ts = REPORT.time_series || [];
const labels = ts.map(b => new Date(b.timestamp * 1000).toLocaleTimeString());
const rpsData = ts.map(b => b.rps);
const p95Data = ts.map(b => b.p95_ms);
const errData = ts.map(b => b.error_rate_pct);

const chartDefaults = {
  type: "line",
  options: {
    responsive: true, maintainAspectRatio: false, animation: false,
    plugins: { legend: { display: false } },
    scales: {
      x: { ticks: { color: "#718096", maxTicksLimit: 10 }, grid: { color: "#2d3748" } },
      y: { ticks: { color: "#718096" }, grid: { color: "#2d3748" } }
    }
  }
};

function mkChart(id, data, color, unit) {
  new Chart(document.getElementById(id), {
    ...chartDefaults,
    data: {
      labels,
      datasets: [{
        data,
        borderColor: color, backgroundColor: color + "22",
        fill: true, tension: 0.3, pointRadius: 0, borderWidth: 2
      }]
    },
    options: { ...chartDefaults.options, plugins: { ...chartDefaults.options.plugins,
      tooltip: { callbacks: { label: ctx => ctx.parsed.y.toFixed(1) + " " + unit } }
    }}
  });
}

mkChart("rpsChart", rpsData, "#63b3ed", "req/s");
mkChart("latencyChart", p95Data, "#68d391", "ms");
mkChart("errorChart", errData, "#fc8181", "%");

// Latency distribution bar chart
const lat = summary.latency;
new Chart(document.getElementById("latencyDistChart"), {
  type: "bar",
  data: {
    labels: ["p50", "p75", "p90", "p95", "p99"],
    datasets: [{
      data: [lat.p50_ms, lat.p75_ms, lat.p90_ms, lat.p95_ms, lat.p99_ms],
      backgroundColor: ["#63b3ed", "#68d391", "#f6ad55", "#fc8181", "#e53e3e"],
    }]
  },
  options: {
    responsive: true, maintainAspectRatio: false, animation: false,
    plugins: { legend: { display: false } },
    scales: {
      x: { ticks: { color: "#718096" }, grid: { color: "#2d3748" } },
      y: { ticks: { color: "#718096", callback: v => v + "ms" }, grid: { color: "#2d3748" } }
    }
  }
});

// Thresholds table
function renderTable(el, headers, rows) {
  let html = '<table><thead><tr>' + headers.map(h => '<th>' + h + '</th>').join('') + '</tr></thead><tbody>';
  rows.forEach(r => { html += '<tr>' + r.map(c => '<td>' + c + '</td>').join('') + '</tr>'; });
  html += '</tbody></table>';
  el.innerHTML = html;
}

const threshEl = document.getElementById("thresholds-table");
if (REPORT.thresholds && REPORT.thresholds.length) {
  renderTable(threshEl,
    ["Metric", "Expression", "Status", "Actual", "Threshold"],
    REPORT.thresholds.map(t => [
      '<code>' + t.name + '</code>',
      t.expression,
      t.passed ? '<span class="badge badge-pass">✓ PASS</span>' : '<span class="badge badge-fail">✗ FAIL</span>',
      fmtActual(t.actual, t.unit),
      fmtActual(t.threshold, t.unit),
    ])
  );
} else {
  threshEl.innerHTML = '<p style="color:#718096">No thresholds defined.</p>';
}

// Scenarios table
const scenEl = document.getElementById("scenarios-table");
if (REPORT.scenarios && REPORT.scenarios.length) {
  renderTable(scenEl,
    ["Scenario", "Requests", "Failures", "Fail %", "avg", "p50", "p95", "p99"],
    REPORT.scenarios.map(s => [
      s.name,
      s.requests.toLocaleString(),
      s.failures.toLocaleString(),
      s.failure_rate_pct.toFixed(2) + "%",
      s.latency.avg_ms.toFixed(0) + "ms",
      s.latency.p50_ms.toFixed(0) + "ms",
      s.latency.p95_ms.toFixed(0) + "ms",
      s.latency.p99_ms.toFixed(0) + "ms",
    ])
  );
} else {
  scenEl.innerHTML = '<p style="color:#718096">No scenarios data.</p>';
}

// Status codes table
const statEl = document.getElementById("status-table");
const codes = REPORT.status_codes || {};
const codeKeys = Object.keys(codes).sort();
if (codeKeys.length) {
  const total = summary.total_requests;
  renderTable(statEl,
    ["Status Code", "Count", "Percentage"],
    codeKeys.map(code => {
      const cnt = codes[code];
      const pct = (cnt / total * 100).toFixed(2);
      const cls = code >= "500" ? "badge-5xx" : code >= "400" ? "badge-4xx" : "badge-2xx";
      return ['<span class="badge ' + cls + '">' + code + '</span>', cnt.toLocaleString(), pct + "%"];
    })
  );
} else {
  statEl.innerHTML = '<p style="color:#718096">No status code data.</p>';
}

function fmtBytes(b) {
  if (b >= 1073741824) return (b/1073741824).toFixed(1) + " GB";
  if (b >= 1048576) return (b/1048576).toFixed(1) + " MB";
  if (b >= 1024) return (b/1024).toFixed(1) + " kB";
  return b + " B";
}

function fmtActual(v, unit) {
  switch(unit) {
    case "ms": return v.toFixed(0) + "ms";
    case "s": return v.toFixed(2) + "s";
    case "%": return v.toFixed(2) + "%";
    default: return v.toFixed(2);
  }
}
</script>
</body>
</html>`
