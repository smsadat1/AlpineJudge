package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	RunCfg RunnerConfig
)

type SchedulerConfig struct {
	OverSubFactor uint8   `yaml:"over_sub_factor"`
	MemResPerc    float32 `yaml:"memory_reserve_percent"`
}

type LimitsConfig struct {
	MemoryLimitMB uint64 `yaml:"memory_limit_mb"`
	PIDLimit      uint16 `yaml:"pid_limit"`
	CPUQuota      uint16 `yaml:"cpu_quota"`
	NoNewPrivs    bool   `yaml:"no_new_privileges"`
	RORootFS      bool   `yaml:"readonly_rootfs"`
	TimeoutSec    uint64 `yaml:"timeout_sec"`
}

type RunnerConfig struct {
	RunnerName string   `yaml:"name"`
	RunnerID   string   `yaml:"runner_id"`
	Images     []string `yaml:"images"`

	Scheduler SchedulerConfig `yaml:"scheduler"`
	Limits    LimitsConfig    `yaml:"limits"`
}

func LoadRunnerConfigs(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("Failed to open config(%v): %v", configPath, err)
	}

	if err := yaml.Unmarshal(data, &RunCfg); err != nil {
		return fmt.Errorf("failed to unmarshal config (%s): %w", configPath, err)
	}

	return nil
}
