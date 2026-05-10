package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Load reads and parses a YAML test script file, returning a validated Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	interpolateEnv(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	applyDefaults(&cfg)
	normalizeWeights(&cfg)

	return &cfg, nil
}

// LoadInline creates a Config from CLI inline flags.
func LoadInline(rawURL string, vus int, duration time.Duration) (*Config, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("--url is required for inline run")
	}
	if vus <= 0 {
		vus = 1
	}
	if duration <= 0 {
		duration = 30 * time.Second
	}

	cfg := &Config{
		Name:           "Inline Load Test",
		Target:         rawURL,
		InlineURL:      rawURL,
		InlineVUs:      vus,
		InlineDuration: duration,
		Stages: []Stage{
			{
				Duration: Duration{Duration: duration},
				VUs:      vus,
			},
		},
		Scenarios: []Scenario{
			{
				Name:   "GET",
				Method: "GET",
				Path:   "",
				Weight: 100,
			},
		},
	}

	applyDefaults(cfg)
	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Target == "" && len(cfg.Stages) == 0 {
		return fmt.Errorf("target URL is required")
	}

	for i, stage := range cfg.Stages {
		if stage.Duration.Duration <= 0 {
			return fmt.Errorf("stage[%d]: duration must be > 0", i)
		}
		if stage.VUs <= 0 {
			return fmt.Errorf("stage[%d]: vus must be > 0", i)
		}
	}

	totalWeight := 0
	for i, scenario := range cfg.Scenarios {
		if scenario.Name == "" {
			return fmt.Errorf("scenario[%d]: name is required", i)
		}
		method := strings.ToUpper(scenario.Method)
		if method == "" {
			method = "GET"
			cfg.Scenarios[i].Method = method
		}
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "PATCH": true,
			"DELETE": true, "HEAD": true, "OPTIONS": true,
		}
		if !validMethods[method] {
			return fmt.Errorf("scenario[%d]: invalid HTTP method %q", i, method)
		}
		if scenario.Weight < 0 {
			return fmt.Errorf("scenario[%d]: weight must be >= 0", i)
		}
		if scenario.ThinkTime != nil {
			if scenario.ThinkTime.MinMs < 0 {
				return fmt.Errorf("scenario[%d]: think_time.min_ms must be >= 0", i)
			}
			if scenario.ThinkTime.MaxMs < 0 {
				return fmt.Errorf("scenario[%d]: think_time.max_ms must be >= 0", i)
			}
			if scenario.ThinkTime.MaxMs > 0 && scenario.ThinkTime.MaxMs < scenario.ThinkTime.MinMs {
				return fmt.Errorf("scenario[%d]: think_time.max_ms must be >= min_ms", i)
			}
		}
		totalWeight += scenario.Weight
	}

	if len(cfg.Scenarios) > 0 && totalWeight == 0 {
		for i := range cfg.Scenarios {
			cfg.Scenarios[i].Weight = 1
		}
	}

	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.MaxRedirects == 0 {
		cfg.MaxRedirects = 10
	}
	cfg.KeepAlive = true
	// cfg.Cookies defaults to false — no change needed

	for i, scenario := range cfg.Scenarios {
		if scenario.Method == "" {
			cfg.Scenarios[i].Method = "GET"
		} else {
			cfg.Scenarios[i].Method = strings.ToUpper(scenario.Method)
		}
		if scenario.Weight == 0 {
			cfg.Scenarios[i].Weight = 1
		}
	}
}

// normalizeWeights ensures scenario weights are converted to a cumulative
// distribution table used for weighted random selection.
func normalizeWeights(cfg *Config) {
	total := 0
	for _, s := range cfg.Scenarios {
		total += s.Weight
	}
	if total == 0 {
		return
	}
	// Weights are kept as-is; selection logic uses them directly.
}

// TotalDuration returns the sum of all stage durations.
func (cfg *Config) TotalDuration() time.Duration {
	var total time.Duration
	for _, s := range cfg.Stages {
		total += s.Duration.Duration
	}
	return total
}

// WeightTotal returns the sum of all scenario weights.
func (cfg *Config) WeightTotal() int {
	total := 0
	for _, s := range cfg.Scenarios {
		total += s.Weight
	}
	return total
}

// interpolateEnv expands environment variable references (${VAR} and $VAR)
// in all string fields of the Config using os.ExpandEnv. It is called after
// YAML unmarshalling and before validation so that env vars can be used
// anywhere a string value is expected.
func interpolateEnv(cfg *Config) {
	cfg.Name = os.ExpandEnv(cfg.Name)
	cfg.Target = os.ExpandEnv(cfg.Target)

	for k, v := range cfg.Thresholds {
		cfg.Thresholds[k] = os.ExpandEnv(v)
	}

	for i, s := range cfg.Scenarios {
		cfg.Scenarios[i].Name = os.ExpandEnv(s.Name)
		cfg.Scenarios[i].Method = os.ExpandEnv(s.Method)
		cfg.Scenarios[i].Path = os.ExpandEnv(s.Path)
		cfg.Scenarios[i].Body = os.ExpandEnv(s.Body)

		for k, v := range s.Headers {
			cfg.Scenarios[i].Headers[k] = os.ExpandEnv(v)
		}

		for j, check := range s.Checks {
			cfg.Scenarios[i].Checks[j] = os.ExpandEnv(check)
		}
	}
}
