// Package config provides YAML test script parsing and validation for bacot.
package config

import "time"

// Config is the top-level structure for a bacot test script.
type Config struct {
	Name       string            `yaml:"name"`
	Target     string            `yaml:"target"`
	Stages     []Stage           `yaml:"stages"`
	Scenarios  []Scenario        `yaml:"scenarios"`
	Thresholds map[string]string `yaml:"thresholds"`

	// Inline run options (set from CLI flags, not YAML)
	InlineURL      string
	InlineVUs      int
	InlineDuration time.Duration

	// HTTP options
	Timeout        time.Duration `yaml:"timeout"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	Insecure       bool          `yaml:"insecure"`
	MaxRedirects   int           `yaml:"max_redirects"`
	HTTP2          bool          `yaml:"http2"`
	KeepAlive      bool          `yaml:"keep_alive"`
	Cookies        bool          `yaml:"cookies"`
}

// Stage defines a load stage with a duration and number of virtual users.
// When StartVUs is set (> 0 and != VUs), the engine linearly interpolates
// the VU count from StartVUs to VUs over the stage duration (gradual ramp).
type Stage struct {
	Duration Duration `yaml:"duration"`
	VUs      int      `yaml:"vus"`
	StartVUs int      `yaml:"start_vus"` // Optional: starting VU count for linear ramp
}

// ThinkTime defines a pause inserted after each request in a scenario.
// If only MinMs is set (MaxMs == 0), the pause is constant.
// If both are set, the pause is uniformly random in [MinMs, MaxMs].
type ThinkTime struct {
	MinMs int `yaml:"min_ms"`
	MaxMs int `yaml:"max_ms"`
}

// Scenario defines a single HTTP request scenario within a test.
type Scenario struct {
	Name      string            `yaml:"name"`
	Method    string            `yaml:"method"`
	Path      string            `yaml:"path"`
	Weight    int               `yaml:"weight"`
	Headers   map[string]string `yaml:"headers"`
	Body      string            `yaml:"body"`
	Checks    []string          `yaml:"checks"`
	ThinkTime *ThinkTime        `yaml:"think_time"` // Optional pause after each request
}

// Duration is a wrapper around time.Duration that supports YAML unmarshaling
// from strings like "30s", "1m30s".
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler for Duration.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

// MarshalYAML implements yaml.Marshaler for Duration.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}
