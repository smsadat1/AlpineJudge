package shared

type JobSpec struct {
	Language             string `json:"language"`
	Version              string `json:"version"`
	Image                string `json:"image"`
	Source               string `json:"source"`
	SubmissionID         string `json:"submission_id"`
	Bucket               string `json:"s3_bucket"`
	SrcCodeS3Key         string `json:"src_code_s3key"`
	TestsetS3Key         string `json:"testset_s3Key"`
	Testset              string `json:"testset"`
	TestsetVersion       string `json:"testset_version"`
	SSEQueue             string `json:"sse_queue"`
	PerTestMemoryLimitMB uint64 `json:"per_test_memory_limit_mb"`
	PerTestTimeoutsec    uint32 `json:"per_test_timeout_sec"`
	PerTestLogLimitKB    uint32 `json:"per_test_log_limit_kb"`
}
