package dispatcher

type SubmissionSpec struct {
	SubmissionID   string `json:"submission_id"`
	Language       string `json:"language"`
	Version        string `json:"version"`
	Source         string `json:"source"`
	Testset        string `json:"testset"`
	TestsetVersion string `json:"testset_version"`
}

type JobSpec struct {
	JobId          string `json:"job_id"`
	SubmissionID   string `json:"submission_id"`
	Language       string `json:"language"`
	Version        string `json:"version"`
	S3Key          string `json:"s3key"`
	Testset        string `json:"testset"`
	TestsetVersion string `json:"testset_version"`
}

func IsLanguageSupported(lang, version string) bool {
	versions, ok := AvailableLanguages[lang]
	if !ok {
		return false
	}

	_, ok = versions[version]
	return ok
}
