package internal

// IMPROTANT: changes, addition or deletion of any field here should be copied to shared module's test's SubmissionSpec_Copy
type SubmissionSpec struct {
	Language             string `json:"language"`
	Source               string `json:"source"`
	SubmissionID         string `json:"submission_id"`
	Bucket               string `json:"bucket"`
	Testset              string `json:"testset_id"`
	PerTestMemoryLimitMB uint64 `json:"memory_limit_mb"`
	PerTestTimeoutsec    uint32 `json:"timeout_sec"`
	PerTestLogLimitKB    uint32 `json:"log_limit_kb"`
}
