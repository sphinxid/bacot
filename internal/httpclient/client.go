// Package httpclient provides HTTP client creation and request execution for bacot.
package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// Options configures the HTTP client behavior.
type Options struct {
	Timeout        time.Duration
	ConnectTimeout time.Duration
	Insecure       bool
	MaxRedirects   int
	HTTP2          bool
	KeepAlive      bool
}

// New creates a new *http.Client configured with the given options.
func New(opts Options) *http.Client {
	dialer := &net.Dialer{
		Timeout:   opts.ConnectTimeout,
		KeepAlive: 30 * time.Second,
	}
	if !opts.KeepAlive {
		dialer.KeepAlive = -1
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: opts.Insecure, //nolint:gosec
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSClientConfig:       tlsCfg,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     !opts.KeepAlive,
	}

	if opts.HTTP2 {
		if err := http2.ConfigureTransport(transport); err == nil {
			_ = err
		}
	}

	redirectPolicy := func(req *http.Request, via []*http.Request) error {
		if opts.MaxRedirects == 0 {
			return http.ErrUseLastResponse
		}
		if len(via) >= opts.MaxRedirects {
			return http.ErrUseLastResponse
		}
		return nil
	}

	return &http.Client{
		Transport:     transport,
		CheckRedirect: redirectPolicy,
		Timeout:       opts.Timeout,
	}
}
