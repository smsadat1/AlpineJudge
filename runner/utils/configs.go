package utils

var (
	RunCfg RunnerConfig
)

type SchedulerConfig struct {
	OverSubFactor uint8   `yaml:"over_sub_factor"`
	MemResPerc    float32 `yaml:"memory_reserve_percent"`
}

type LimitsConfig struct {
	MemoryLimitMB uint64 `yaml:"memory_limit_mb"`
	PIDLimit      int64  `yaml:"pid_limit"`
	CPUQuota      uint16 `yaml:"cpu_quota"`
	NoNewPrivs    bool   `yaml:"no_new_privileges"`
	RORootFS      bool   `yaml:"readonly_rootfs"`
	TimeoutSec    uint64 `yaml:"timeout_sec"`
	LogLimitKB    uint64 `yaml:"log_limit_kb"`
}

type RunnerConfig struct {
	RunnerID string            `yaml:"runner_id"`
	Images   map[string]string `yaml:"images"`

	Scheduler SchedulerConfig `yaml:"scheduler"`
	Limits    LimitsConfig    `yaml:"limits"`
}
