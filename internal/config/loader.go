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
