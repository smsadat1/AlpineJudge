package dispatcher

type SubmissionSpec struct {
	SubmissionID   string `json:"submission_id"`
	Bucket         string `json:"bucket"`
	Language       string `json:"language"`
	Version        string `json:"version"`
	Source         string `json:"source"`
	Testset        string `json:"testset"`
	TestsetVersion string `json:"testset_version"`
}
