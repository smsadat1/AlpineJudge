package utils

type ExecRules struct {
	// system
	ContainerID string
	Image       string
	Args        []string

	// environment
	HostSrcpath       string
	ContainerDestPath string
	Env               map[string]string

	// rules
	MemoryLimitMB  uint64
	PidLimit       int64
	CpuQuota       float64
	NoNewPrivilege bool
	ReadOnlyRootfs bool
	Timeoutsec     uint32
}

type Verdict string

const (
	VerdictAC  Verdict = "AC"
	VerdictWA  Verdict = "WA"
	VerdictTLE Verdict = "TLE"
	VerdictMLE Verdict = "MLE"
	VerdictOLE Verdict = "OLE"
	VerdictCE  Verdict = "CE"
	VerdictRE  Verdict = "RE"
	VerdictIE  Verdict = "IE"
	VerdictPE  Verdict = "PE"
	VerdictSE  Verdict = "SE"
)
