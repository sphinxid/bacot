package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/sphinxid/bacot/internal/config"
	"github.com/sphinxid/bacot/internal/engine"
	"github.com/sphinxid/bacot/internal/metrics"
	"github.com/sphinxid/bacot/internal/output"
	"github.com/sphinxid/bacot/internal/thresholds"
	"github.com/sphinxid/bacot/internal/version"
)

var (
	flagURL            string
	flagVUs            int
	flagDuration       string
	flagTimeout        string
	flagConnectTimeout string
	flagInsecure       bool
	flagMaxRedirects   int
	flagHTTP2          bool
)

var runCmd = &cobra.Command{
	Use:   "run [script.yaml]",
	Short: "Run a load test from a YAML script or inline flags",
	Long: `Run a load test from a YAML script file, or use inline flags for a quick test.

Examples:
  bacot run script.yaml
  bacot run --url https://example.com --vus 10 --duration 30s
  bacot run script.yaml --output json=result.json --output html=report.html`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTest,
}

func init() {
	runCmd.Flags().StringVar(&flagURL, "url", "", "target URL for inline test")
	runCmd.Flags().IntVar(&flagVUs, "vus", 1, "number of virtual users for inline test")
	runCmd.Flags().StringVar(&flagDuration, "duration", "30s", "test duration for inline test (e.g. 30s, 1m)")
	runCmd.Flags().StringVar(&flagTimeout, "timeout", "30s", "HTTP request timeout")
	runCmd.Flags().StringVar(&flagConnectTimeout, "connect-timeout", "10s", "TCP connect timeout")
	runCmd.Flags().BoolVar(&flagInsecure, "insecure", false, "skip TLS certificate verification")
	runCmd.Flags().IntVar(&flagMaxRedirects, "max-redirects", 10, "maximum number of redirects to follow")
	runCmd.Flags().BoolVar(&flagHTTP2, "http2", false, "enable HTTP/2 support")
}

func runTest(cmd *cobra.Command, args []string) error {
	var cfg *config.Config
	var err error

	if len(args) == 1 {
		cfg, err = config.Load(args[0])
		if err != nil {
			return fmt.Errorf("loading script: %w", err)
		}
	} else {
		duration, err := time.ParseDuration(flagDuration)
		if err != nil {
			return fmt.Errorf("parsing --duration: %w", err)
		}
		cfg, err = config.LoadInline(flagURL, flagVUs, duration)
		if err != nil {
			return err
		}
	}

	// Apply CLI HTTP options (override YAML if explicitly set)
	if timeout, err := time.ParseDuration(flagTimeout); err == nil && timeout > 0 {
		cfg.Timeout = timeout
	}
	if ct, err := time.ParseDuration(flagConnectTimeout); err == nil && ct > 0 {
		cfg.ConnectTimeout = ct
	}
	if flagInsecure {
		cfg.Insecure = true
	}
	if flagMaxRedirects != 10 {
		cfg.MaxRedirects = flagMaxRedirects
	}
	if flagHTTP2 {
		cfg.HTTP2 = true
	}

	// Print header
	boldCyan := color.New(color.Bold, color.FgCyan)
	fmt.Fprintf(os.Stderr, "\n%s\n\n",
		boldCyan.Sprintf("bacot v%s — %s", version.Version, cfg.Name))

	startedAt := time.Now()
	collector := metrics.NewCollector()
	eng := engine.New(cfg, collector)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	totalDuration := cfg.TotalDuration()
	stageCount := len(cfg.Stages)

	var dashboard *output.Dashboard
	if !quietFlag {
		dashboard = output.NewDashboard(os.Stderr, cfg, collector, eng, noColorFlag)
	}

	// Render ticker
	var tickerDone chan struct{}
	if !quietFlag {
		tickerDone = make(chan struct{})
		go func() {
			defer close(tickerDone)
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					dashboard.Clear()
					return
				case <-ticker.C:
					elapsed := time.Since(startedAt)
					dashboard.Render(stageCount, elapsed, totalDuration)
				}
			}
		}()
	}

	// Run signal handler
	go func() {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\n  Interrupted — stopping gracefully…")
			cancel()
		case <-ctx.Done():
		}
	}()

	// Run the engine (blocks until done)
	eng.Run(ctx)

	// Stop the dashboard ticker
	if !quietFlag {
		cancel()
		<-tickerDone
	}

	elapsed := time.Since(startedAt)

	// Evaluate thresholds
	latSnap := collector.Latency.Snapshot()
	snap := thresholds.MetricsSnapshot{
		DurationP50Micros: latSnap.P50,
		DurationP75Micros: latSnap.P75,
		DurationP90Micros: latSnap.P90,
		DurationP95Micros: latSnap.P95,
		DurationP99Micros: latSnap.P99,
		DurationAvgMicros: latSnap.Mean,
		DurationMinMicros: latSnap.Min,
		DurationMaxMicros: latSnap.Max,
		FailureRate:       collector.FailureRate(),
		TotalRequests:     collector.TotalRequests.Load(),
		TotalFailures:     collector.TotalFailures.Load(),
		RPS:               collector.RPS(),
	}

	var thresholdResults []thresholds.Result
	allPassed := true
	if len(cfg.Thresholds) > 0 {
		thresholdResults, allPassed = thresholds.EvaluateAll(cfg.Thresholds, snap)
	}

	// Print summary
	output.PrintSummary(os.Stdout, collector, thresholdResults, elapsed)

	// Print full per-scenario report if requested
	if fullReportFlag {
		output.PrintScenarioReport(os.Stdout, collector)
	}

	// Write reports
	outputs := parseOutputFlag(outputFlag)
	if jsonPath, ok := outputs["json"]; ok {
		if err := output.WriteJSON(jsonPath, cfg.Name, collector, thresholdResults, elapsed, startedAt); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "  JSON report written to: %s\n", jsonPath)
		}
	}
	if htmlPath, ok := outputs["html"]; ok {
		if err := output.WriteHTML(htmlPath, cfg.Name, collector, thresholdResults, elapsed, startedAt); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "  HTML report written to: %s\n", htmlPath)
		}
	}

	if !allPassed {
		os.Exit(1)
	}
	return nil
}
