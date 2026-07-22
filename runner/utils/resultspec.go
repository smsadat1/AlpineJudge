package utils

type Verdict string

const (
	VerdictAC  Verdict = "AC"  // accepted
	VerdictWA  Verdict = "WA"  // wrong answer
	VerdictTLE Verdict = "TLE" // time limit exceeded
	VerdictMLE Verdict = "MLE" // memeory limit exceeded
	VerdictOLE Verdict = "OLE" // output limit exceeded
	VerdictCE  Verdict = "CE"  // compilation error
	VerdictRE  Verdict = "RE"  // runtime error
	VerdictIE  Verdict = "IE"  // internal error
	VerdictPE  Verdict = "PE"  // presentation error
	VerdictSE  Verdict = "SE"  // security error
)

type ResultSpec struct {
	SubmissionId string  `json:"submission_id"`
	Language     string  `json:"language"`
	Version      string  `json:"version"`
	Interval     uint64  `json:"interval"`
	Status       Verdict `json:"status"`
	Details      string  `json:"details"`
}
