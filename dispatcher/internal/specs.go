package internal

type SubmissionSpec struct {
	SubmissionID string `json:"submission_id"`
	Bucket       string `json:"bucket"`
	Language     string `json:"language"`
	Source       string `json:"source"`
	Testset      string `json:"testset_id"`
}
