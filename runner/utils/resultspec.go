package utils

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

type ResultSpec struct {
	SubmissionId string  `json:"submission_id"`
	Language     string  `json:"language"`
	Version      string  `json:"version"`
	Interval     string  `json:"interval"`
	Status       Verdict `json:"status"`
}
