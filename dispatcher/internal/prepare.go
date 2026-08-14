package internal

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"shared"
)

var availableLanguages []string = []string{"c", "cpp", "java", "python", "go", "js"}

func ValidateSubmission(ctx context.Context, s3m shared.S3Manager, submission SubmissionSpec) error {

	dirPath := submission.SubmissionID
	language := submission.Language
	testset := submission.Testset

	// check submission_id uniqueness
	exists, err := s3m.CheckS3Dir(ctx, dirPath)
	if err != nil {
		return err
	}

	if exists {
		return fmt.Errorf("submission ID already used")
	}

	// check language availability
	if !slices.Contains(availableLanguages, submission.Language) {
		return fmt.Errorf("Unsupported language or version %v", language)
	}

	// check testset & testsetVer
	ok, err := s3m.CheckS3Dir(ctx, submission.Testset)

	if !ok {
		return fmt.Errorf("Testset: %v not found in S3", testset)
	}
	if err != nil {
		return err
	}

	return nil
}

func PrepareSubmission(
	ctx context.Context, s3m shared.S3Manager, submission SubmissionSpec,
) (shared.JobSpec, error) {
	source := submission.Source
	body := strings.NewReader(source)
	srcS3key := "submissions/" + submission.SubmissionID + "/"
	testS3key := submission.Testset + "/"

	if err := s3m.UploadFileToS3(ctx, srcS3key, body); err != nil {
		return shared.JobSpec{}, err
	}

	jobspec := shared.JobSpec{
		SubmissionID: submission.SubmissionID,
		Language:     submission.Language,
		Bucket:       submission.Bucket,
		SrcCodeS3Key: srcS3key,
		TestsetS3Key: testS3key,
		Testset:      submission.Testset,
	}

	return jobspec, nil
}
