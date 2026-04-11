// Package output provides terminal rendering, summary printing, and report generation.
package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/sphinxid/bacot/internal/config"
	"github.com/sphinxid/bacot/internal/metrics"
)

// Dashboard renders a live refreshing terminal dashboard for the load test.
type Dashboard struct {
	out       io.Writer
	cfg       *config.Config
	collector *metrics.Collector
	engine    ActiveVUsProvider
	noColor   bool
	lineCount int
}

// ActiveVUsProvider is implemented by the engine to expose current VU count and stage.
type ActiveVUsProvider interface {
	ActiveVUs() int64
	CurrentStage() int64
}

// NewDashboard creates a new Dashboard.
func NewDashboard(out io.Writer, cfg *config.Config, collector *metrics.Collector, engine ActiveVUsProvider, noColor bool) *Dashboard {
	if noColor {
		color.NoColor = true
	}
	return &Dashboard{
		out:       out,
		cfg:       cfg,
		collector: collector,
		engine:    engine,
		noColor:   noColor,
	}
}

var (
	boldW  = color.New(color.Bold, color.FgWhite)
	cyan   = color.New(color.FgCyan)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
	yellow = color.New(color.FgYellow)
	dim    = color.New(color.Faint)
)

// Render clears the previous render and draws the current state.
func (d *Dashboard) Render(stageCount int, elapsed, totalDuration time.Duration) {
	d.clearLines()

	lines := d.buildLines(stageCount, elapsed, totalDuration)
	for _, line := range lines {
		fmt.Fprintln(d.out, line)
	}
	d.lineCount = len(lines)
}

// Clear removes the dashboard from the terminal.
func (d *Dashboard) Clear() {
	d.clearLines()
	d.lineCount = 0
}

func (d *Dashboard) clearLines() {
	for i := 0; i < d.lineCount; i++ {
		fmt.Fprint(d.out, "\033[1A\033[2K")
	}
}

func (d *Dashboard) buildLines(stageCount int, elapsed, totalDuration time.Duration) []string {
	var lines []string

	sep := dim.Sprint(strings.Repeat("─", 58))
	lines = append(lines, sep)

	// Stage progress
	stageIdx := d.engine.CurrentStage() + 1
	progressBar := renderProgressBar(elapsed, totalDuration, 20)
	stageStr := fmt.Sprintf("  Stage %d/%d  %s  %s / %s",
		stageIdx, stageCount,
		cyan.Sprint(progressBar),
		formatDuration(elapsed),
		formatDuration(totalDuration),
	)
	lines = append(lines, stageStr)

	activeVUs := d.engine.ActiveVUs()
	lines = append(lines, fmt.Sprintf("  VUs: %s active", boldW.Sprintf("%d", activeVUs)))
	lines = append(lines, "")

	// Metrics
	total := d.collector.TotalRequests.Load()
	failures := d.collector.TotalFailures.Load()
	rps := d.collector.RPS()
	failPct := float64(0)
	if total > 0 {
		failPct = float64(failures) / float64(total) * 100
	}

	latSnap := d.collector.Latency.Snapshot()
	p50ms := float64(latSnap.P50) / 1000.0
	p95ms := float64(latSnap.P95) / 1000.0
	p99ms := float64(latSnap.P99) / 1000.0

	lines = append(lines, fmt.Sprintf("  Requests      %s   RPS: %s/s",
		boldW.Sprintf("%s", formatInt(total)),
		boldW.Sprintf("%.1f", rps),
	))

	failColor := green
	if failPct > 1 {
		failColor = red
	} else if failPct > 0 {
		failColor = yellow
	}
	lines = append(lines, fmt.Sprintf("  Failures      %s   %s",
		failColor.Sprintf("%s", formatInt(failures)),
		dim.Sprintf("(%.2f%%)", failPct),
	))

	lines = append(lines, fmt.Sprintf("  Duration      p50: %s   p95: %s   p99: %s",
		cyan.Sprintf("%.0fms", p50ms),
		cyan.Sprintf("%.0fms", p95ms),
		cyan.Sprintf("%.0fms", p99ms),
	))

	sent := d.collector.TotalBytesSent.Load()
	recv := d.collector.TotalBytesRecv.Load()
	lines = append(lines, fmt.Sprintf("  Data          %s %s   %s %s",
		yellow.Sprint("↑"), formatBytes(sent),
		cyan.Sprint("↓"), formatBytes(recv),
	))

	checksPassed := d.collector.ChecksPassed.Load()
	checksFailed := d.collector.ChecksFailed.Load()
	if checksPassed+checksFailed > 0 {
		lines = append(lines, fmt.Sprintf("  Checks        %s %s   %s %s",
			green.Sprint("✓"), green.Sprintf("%s", formatInt(checksPassed)),
			red.Sprint("✗"), red.Sprintf("%s", formatInt(checksFailed)),
		))
	}

	lines = append(lines, "")
	lines = append(lines, sep)

	// Per-scenario breakdown
	scenarios := d.collector.ScenarioSnapshot()
	if len(scenarios) > 0 {
		lines = append(lines, boldW.Sprint("  Scenarios"))
		for i, sm := range scenarios {
			prefix := "  ├─"
			if i == len(scenarios)-1 {
				prefix = "  └─"
			}
			reqs := sm.Requests.Load()
			smFail := sm.Failures.Load()
			smP95 := float64(sm.Latency.Percentile(95)) / 1000.0
			okPct := float64(100)
			if reqs > 0 {
				okPct = float64(reqs-smFail) / float64(reqs) * 100
			}
			okColor := green
			if okPct < 99 {
				okColor = red
			}
			lines = append(lines, fmt.Sprintf("%s %-20s %s reqs   p95: %s   %s",
				dim.Sprint(prefix),
				sm.Name,
				formatInt(reqs),
				cyan.Sprintf("%.0fms", smP95),
				okColor.Sprintf("✓ %.1f%%", okPct),
			))
		}
	}

	return lines
}

// renderProgressBar returns an ASCII progress bar string.
func renderProgressBar(elapsed, total time.Duration, width int) string {
	if total <= 0 {
		return "[" + strings.Repeat("░", width) + "]"
	}
	pct := float64(elapsed) / float64(total)
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * float64(width))
	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
	return bar
}

// formatDuration formats a duration as a short human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", mins, secs)
}

// formatInt formats an integer with thousands separators.
func formatInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	result := make([]byte, 0, len(s)+len(s)/3)
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f kB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
