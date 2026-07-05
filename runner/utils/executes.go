package utils

type ExecRules struct {
	// system
	ContainerID string
	Image       string
	Args        []string

	// environment
	CodePathHost         string
	CodePathContainer    string
	TestsetPathHost      string
	TestsetPathContainer string
	Env                  map[string]string

	// rules
	MemoryLimitMB  uint64
	PidLimit       int64
	CpuQuota       float64
	NoNewPrivilege bool
	ReadOnlyRootfs bool
	Timeoutsec     uint32
}
