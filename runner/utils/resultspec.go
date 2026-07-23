package utils

type Verdict string

const (
	VerdictAC  Verdict = "AC"  // accepted (agent)
	VerdictWA  Verdict = "WA"  // wrong answer (agent)
	VerdictTLE Verdict = "TLE" // time limit exceeded (container)
	VerdictMLE Verdict = "MLE" // memeory limit exceeded (container)
	VerdictOLE Verdict = "OLE" // output limit exceeded (agent)
	VerdictCE  Verdict = "CE"  // compilation error (agent)
	VerdictRE  Verdict = "RE"  // runtime error (agent)
	VerdictIE  Verdict = "IE"  // internal error (agent + container)
	VerdictPV  Verdict = "PV"  // policy violation (container)
	VerdictSE  Verdict = "SE"  // security error (host)
)

type ResultSpec struct {
	SubmissionId string  `json:"submission_id"`
	Language     string  `json:"language"`
	Version      string  `json:"version"`
	Interval     uint64  `json:"interval"`
	Status       Verdict `json:"status"`
	Details      string  `json:"details"`
}
