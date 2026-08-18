package shared

type JobSpec struct {
	Language             string `json:"language"`
	Source               string `json:"source"`
	SubmissionID         string `json:"submission_id"`
	Bucket               string `json:"s3_bucket"`
	SrcCodeS3Key         string `json:"src_code_s3key"`
	TestsetS3Key         string `json:"testset_s3Key"`
	Testset              string `json:"testset_id"`
	SSEQueue             string `json:"sse_queue"`
	PerTestMemoryLimitMB uint64 `json:"memory_limit_mb"`
	PerTestTimeoutsec    uint32 `json:"timeout_sec"`
	PerTestLogLimitKB    uint32 `json:"log_limit_kb"`
}
