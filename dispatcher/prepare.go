package dispatcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/sixafter/nanoid"

	"shared"
)

func ValidateSubmission(ctx context.Context, s3m shared.S3Manager, submission SubmissionSpec) error {

	dirPath := submission.SubmissionID
	language := submission.Language
	version := submission.Version
	testset := submission.Testset
	testsetVer := submission.TestsetVersion

	// check submission_id uniqueness
	ok, err := s3m.CheckS3Dir(ctx, dirPath)
	if !ok {
		return fmt.Errorf("SubmissionID already used")
	}
	if err != nil {
		return err
	}

	// check language & version availability
	ok = IsLanguageSupported(language, version)
	if !ok {
		return fmt.Errorf("Unsupported language or version [Lang: %v Ver: %v]", language, version)
	}

	// check testset & testsetVer
	ok, err = s3m.CheckS3Dir(ctx, string(testset+"/"+testsetVer))

	if !ok {
		return fmt.Errorf("Testset: [%v/%v] not found in S3", testset, testsetVer)
	}
	if err != nil {
		return err
	}

	return nil
}

func PrepareSubmission(ctx context.Context, s3m shared.S3Manager, submission SubmissionSpec) (JobSpec, error) {

	// generate job_id
	jobID, err := nanoid.New()
	if err != nil {
		return JobSpec{}, err
	}

	source := submission.Source
	body := strings.NewReader(source)
	s3key := submission.SubmissionID + "/" + string(jobID)

	if err := s3m.UploadToS3(ctx, s3key, body); err != nil {
		return JobSpec{}, err
	}

	jobspec := JobSpec{
		JobId:          string(jobID),
		SubmissionID:   submission.SubmissionID,
		Language:       submission.Language,
		Version:        submission.Version,
		S3Key:          s3key,
		Testset:        submission.Testset,
		TestsetVersion: submission.TestsetVersion,
	}

	return jobspec, nil
}
