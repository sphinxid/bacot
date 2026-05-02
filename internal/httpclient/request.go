package httpclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"time"

	"github.com/sphinxid/bacot/internal/metrics"
)

// RequestSpec defines the parameters of an HTTP request to execute.
type RequestSpec struct {
	Method          string
	URL             string
	Headers         map[string]string
	Body            string
	Name            string
	CaptureBody     bool // When true, response body is stored in RequestResult.ResponseBody
	CaptureHeaders  bool // When true, response headers are stored in RequestResult.ResponseHeaders
}

// Execute performs the HTTP request described by spec using the given client.
// It collects detailed timing data and returns a RequestResult.
func Execute(ctx context.Context, client *http.Client, spec RequestSpec) metrics.RequestResult {
	result := metrics.RequestResult{
		ScenarioName: spec.Name,
	}

	var bodyReader io.Reader
	if spec.Body != "" {
		bodyBytes := []byte(spec.Body)
		result.BytesSent = int64(len(bodyBytes))
		// Check if body is a file path
		if strings.HasPrefix(spec.Body, "@") {
			filePath := spec.Body[1:]
			data, err := os.ReadFile(filePath)
			if err != nil {
				result.Error = fmt.Errorf("reading body file %q: %w", filePath, err)
				result.ErrorType = "io"
				return result
			}
			bodyBytes = data
			result.BytesSent = int64(len(bodyBytes))
		}
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, spec.Method, spec.URL, bodyReader)
	if err != nil {
		result.Error = fmt.Errorf("building request: %w", err)
		result.ErrorType = "request"
		return result
	}

	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}

	// Timing via httptrace
	var (
		dnsStart, dnsDone         time.Time
		connectStart, connectDone time.Time
		tlsStart, tlsDone         time.Time
		ttfb                      time.Time
	)

	trace := &httptrace.ClientTrace{
		DNSStart:          func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:           func(_ httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectStart:      func(_, _ string) { connectStart = time.Now() },
		ConnectDone:       func(_, _ string, _ error) { connectDone = time.Now() },
		TLSHandshakeStart: func() { tlsStart = time.Now() },
		TLSHandshakeDone:  func(_ tls.ConnectionState, _ error) { tlsDone = time.Now() },
		GotFirstResponseByte: func() { ttfb = time.Now() },
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.Error = err
		result.ErrorType = classifyError(err)
		result.DurationMicros = time.Since(start).Microseconds()
		return result
	}
	defer resp.Body.Close()

	// Read body: capture it when needed for checks, otherwise discard.
	var n int64
	if spec.CaptureBody {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			result.ResponseBody = bodyBytes
		}
		n = int64(len(bodyBytes))
	} else {
		n, _ = io.Copy(io.Discard, resp.Body)
	}
	total := time.Since(start)

	result.StatusCode = resp.StatusCode
	result.BytesRecv = n
	result.DurationMicros = total.Microseconds()

	if spec.CaptureHeaders {
		hdrs := make(map[string]string, len(resp.Header))
		for k, vals := range resp.Header {
			if len(vals) > 0 {
				hdrs[k] = vals[0]
			}
		}
		result.ResponseHeaders = hdrs
	}

	if !dnsStart.IsZero() && !dnsDone.IsZero() {
		result.ConnectMicros = dnsDone.Sub(dnsStart).Microseconds()
	}
	if !connectStart.IsZero() && !connectDone.IsZero() {
		result.ConnectMicros += connectDone.Sub(connectStart).Microseconds()
	}
	if !tlsStart.IsZero() && !tlsDone.IsZero() {
		result.TLSMicros = tlsDone.Sub(tlsStart).Microseconds()
	}
	if !ttfb.IsZero() {
		result.TTFBMicros = ttfb.Sub(start).Microseconds()
	}

	return result
}

// classifyError categorizes a network/HTTP error into a string type.
func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "timeout"
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Op == "dial" {
			var dnsErr *net.DNSError
			if errors.As(err, &dnsErr) {
				return "dns"
			}
			return "refused"
		}
		return "network"
	}
	return "http"
}
