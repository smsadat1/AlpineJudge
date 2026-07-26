package utils

type Verdict string

const (
	VerdictAC  Verdict = "AC"                    // accepted (agent)
	VerdictWA  Verdict = "WA"                    // wrong answer (agent)
	VerdictTLE Verdict = "Time limit exceeded"   // time limit exceeded (container)
	VerdictMLE Verdict = "Memory limit exceeded" // memeory limit exceeded (container)
	VerdictOLE Verdict = "OLE"                   // output limit exceeded (agent)
	VerdictCE  Verdict = "CE"                    // compilation error (agent)
	VerdictRE  Verdict = "RE"                    // runtime error (agent)
	VerdictIE  Verdict = "IE"                    // internal error (agent + container)
	VerdictPV  Verdict = "PV"                    // policy violation (container)
	VerdictSE  Verdict = "SE"                    // security error (host)
)

type ContainerInfo struct {
	SubmissionId    string  `json:"submission_id"`
	Language        string  `json:"language"`
	Version         string  `json:"version"`
	Interval        uint64  `json:"interval"`
	Status          Verdict `json:"status"`
	StatusInfo      string  `json:"status_info"`
	ContainerStdout string  `json:"container_stdout"`
	ContainerStderr string  `json:"container_stderr"`
}
