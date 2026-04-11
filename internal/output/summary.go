package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"

	"github.com/sphinxid/bacot/internal/metrics"
	"github.com/sphinxid/bacot/internal/thresholds"
)

// PrintSummary writes the final test summary to out.
func PrintSummary(out io.Writer, collector *metrics.Collector, thresholdResults []thresholds.Result, elapsed time.Duration) {
	sep := dim.Sprint(strings.Repeat("─", 58))
	title := boldW.Sprint(strings.Repeat("─", 22) + " SUMMARY " + strings.Repeat("─", 22))

	fmt.Fprintln(out)
	fmt.Fprintln(out, title)
	fmt.Fprintln(out)

	total := collector.TotalRequests.Load()
	failures := collector.TotalFailures.Load()
	sent := collector.TotalBytesSent.Load()
	recv := collector.TotalBytesRecv.Load()
	rps := collector.RPS()
	failPct := float64(0)
	if total > 0 {
		failPct = float64(failures) / float64(total) * 100
	}

	elapsedSec := elapsed.Seconds()
	sentRate := float64(sent) / elapsedSec
	recvRate := float64(recv) / elapsedSec

	lat := collector.Latency.Snapshot()
	avgMs := lat.Mean / 1000.0
	minMs := float64(lat.Min) / 1000.0
	p90Ms := float64(lat.P90) / 1000.0
	p95Ms := float64(lat.P95) / 1000.0
	p99Ms := float64(lat.P99) / 1000.0

	printMetric(out, "http_reqs", fmt.Sprintf("%s   %.1f/s", formatInt(total), rps))
	printMetric(out, "http_req_duration",
		fmt.Sprintf("avg=%.0fms   min=%.0fms   p90=%.0fms   p95=%.0fms   p99=%.0fms",
			avgMs, minMs, p90Ms, p95Ms, p99Ms))

	failStr := fmt.Sprintf("%.2f%%   (%s of %s)", failPct, formatInt(failures), formatInt(total))
	if failPct > 0 {
		printMetricColored(out, "http_req_failed", failStr, red)
	} else {
		printMetricColored(out, "http_req_failed", failStr, green)
	}

	printMetric(out, "data_sent",
		fmt.Sprintf("%s   %s/s", formatBytes(sent), formatBytes(int64(sentRate))))
	printMetric(out, "data_received",
		fmt.Sprintf("%s   %s/s", formatBytes(recv), formatBytes(int64(recvRate))))

	// Checks
	checksPassed := collector.ChecksPassed.Load()
	checksFailed := collector.ChecksFailed.Load()
	totalChecks := checksPassed + checksFailed
	if totalChecks > 0 {
		checkPct := float64(checksPassed) / float64(totalChecks) * 100
		checkStr := fmt.Sprintf("%.1f%%   %s %s / %s %s",
			checkPct,
			green.Sprint("✓"), formatInt(checksPassed),
			red.Sprint("✗"), formatInt(checksFailed),
		)
		printMetric(out, "checks", checkStr)
	}

	fmt.Fprintln(out)

	// Thresholds
	if len(thresholdResults) > 0 {
		fmt.Fprintln(out, boldW.Sprint("  Thresholds"))
		for _, tr := range thresholdResults {
			var line string
			actualStr := formatThresholdActual(tr)
			if tr.Passed {
				line = fmt.Sprintf("  %s %-35s %s",
					green.Sprint("✓"),
					cyan.Sprintf("%s %s", tr.Name, tr.Expression),
					dim.Sprintf("(actual: %s)", actualStr),
				)
			} else {
				line = fmt.Sprintf("  %s %-35s %s",
					red.Sprint("✗"),
					red.Sprintf("%s %s", tr.Name, tr.Expression),
					dim.Sprintf("(actual: %s)", actualStr),
				)
			}
			fmt.Fprintln(out, line)
		}
		fmt.Fprintln(out)
	}

	// Status codes
	statusCodes := collector.StatusCodes.Snapshot()
	if len(statusCodes) > 0 {
		fmt.Fprintln(out, boldW.Sprint("  Status Codes"))
		codes := make([]int, 0, len(statusCodes))
		for code := range statusCodes {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		for i, code := range codes {
			count := statusCodes[code]
			pct := float64(count) / float64(total) * 100
			prefix := "  ├─"
			if i == len(codes)-1 {
				prefix = "  └─"
			}
			codeColor := green
			if code >= 500 {
				codeColor = red
			} else if code >= 400 {
				codeColor = yellow
			}
			fmt.Fprintf(out, "%s %s: %s (%s)\n",
				dim.Sprint(prefix),
				codeColor.Sprintf("%d", code),
				formatInt(count),
				dim.Sprintf("%.2f%%", pct),
			)
		}
		fmt.Fprintln(out)
	}

	fmt.Fprintln(out, sep)
}

// PrintScenarioReport prints a full per-scenario metrics breakdown to out.
// It is called when --full-report is enabled.
func PrintScenarioReport(out io.Writer, collector *metrics.Collector) {
	scenarios := collector.ScenarioSnapshot()
	if len(scenarios) == 0 {
		return
	}

	// Sort by request count descending
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].Requests.Load() > scenarios[j].Requests.Load()
	})

	title := boldW.Sprint(strings.Repeat("─", 18) + " SCENARIO REPORT " + strings.Repeat("─", 18))
	sep := dim.Sprint(strings.Repeat("─", 58))

	fmt.Fprintln(out)
	fmt.Fprintln(out, title)

	total := collector.TotalRequests.Load()

	for i, sm := range scenarios {
		reqs := sm.Requests.Load()
		fails := sm.Failures.Load()
		failPct := float64(0)
		if reqs > 0 {
			failPct = float64(fails) / float64(reqs) * 100
		}
		lat := sm.Latency.Snapshot()
		avgMs := lat.Mean / 1000.0
		minMs := float64(lat.Min) / 1000.0
		p50Ms := float64(lat.P50) / 1000.0
		p75Ms := float64(lat.P75) / 1000.0
		p90Ms := float64(lat.P90) / 1000.0
		p95Ms := float64(lat.P95) / 1000.0
		p99Ms := float64(lat.P99) / 1000.0
		maxMs := float64(lat.Max) / 1000.0

		trafficPct := float64(0)
		if total > 0 {
			trafficPct = float64(reqs) / float64(total) * 100
		}

		failColor := green
		if failPct > 1 {
			failColor = red
		} else if failPct > 0 {
			failColor = yellow
		}

		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %s  %s\n",
			boldW.Sprintf("[%d/%d]", i+1, len(scenarios)),
			boldW.Sprint(sm.Name),
		)
		fmt.Fprintln(out, dim.Sprint("  "+strings.Repeat("·", 54)))

		printMetric(out, "traffic_share", dim.Sprintf("%.1f%% of total requests", trafficPct))
		printMetric(out, "http_reqs", fmt.Sprintf("%s   %.1f/s",
			formatInt(reqs),
			float64(reqs)/collector.Elapsed().Seconds(),
		))

		failStr := fmt.Sprintf("%.2f%%   (%s of %s)", failPct, formatInt(fails), formatInt(reqs))
		if failPct > 0 {
			printMetricColored(out, "http_req_failed", failStr, failColor)
		} else {
			printMetricColored(out, "http_req_failed", failStr, green)
		}

		printMetric(out, "http_req_duration",
			fmt.Sprintf("avg=%.0fms   min=%.0fms   max=%.0fms", avgMs, minMs, maxMs))
		printMetric(out, "  p50/p75/p90",
			cyan.Sprintf("%.0fms / %.0fms / %.0fms", p50Ms, p75Ms, p90Ms))
		printMetric(out, "  p95/p99    ",
			cyan.Sprintf("%.0fms / %.0fms", p95Ms, p99Ms))

		bytesSent := sm.BytesSent.Load()
		bytesRecv := sm.BytesRecv.Load()
		printMetric(out, "data_sent", formatBytes(bytesSent))
		printMetric(out, "data_received", formatBytes(bytesRecv))

		checkOk := sm.ChecksPassed.Load()
		checkFl := sm.ChecksFailed.Load()
		if checkOk+checkFl > 0 {
			checkTotal := checkOk + checkFl
			checkPct := float64(checkOk) / float64(checkTotal) * 100
			checkColor := green
			if checkPct < 100 {
				checkColor = yellow
			}
			if checkPct < 95 {
				checkColor = red
			}
			printMetricColored(out, "checks",
				fmt.Sprintf("%.1f%%   %s %s / %s %s",
					checkPct,
					green.Sprint("✓"), formatInt(checkOk),
					red.Sprint("✗"), formatInt(checkFl),
				), checkColor)
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, sep)
}

func printMetric(out io.Writer, name, value string) {
	dots := 24 - len(name)
	if dots < 1 {
		dots = 1
	}
	fmt.Fprintf(out, "  %s%s: %s\n",
		boldW.Sprint(name),
		dim.Sprint(strings.Repeat(".", dots)),
		value,
	)
}

func printMetricColored(out io.Writer, name, value string, c *color.Color) {
	dots := 24 - len(name)
	if dots < 1 {
		dots = 1
	}
	fmt.Fprintf(out, "  %s%s: %s\n",
		boldW.Sprint(name),
		dim.Sprint(strings.Repeat(".", dots)),
		c.Sprint(value),
	)
}

func formatThresholdActual(tr thresholds.Result) string {
	switch tr.Unit {
	case "ms":
		return fmt.Sprintf("%.0fms", tr.Actual)
	case "s":
		return fmt.Sprintf("%.2fs", tr.Actual)
	case "%":
		return fmt.Sprintf("%.2f%%", tr.Actual)
	default:
		return fmt.Sprintf("%.2f", tr.Actual)
	}
}
