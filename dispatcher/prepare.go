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

func PrepareSubmission(
	ctx context.Context, s3m shared.S3Manager, submission SubmissionSpec,
) (shared.JobSpec, error) {

	// generate job_id
	jobID, err := nanoid.New()
	if err != nil {
		return shared.JobSpec{}, err
	}

	source := submission.Source
	body := strings.NewReader(source)
	srcS3key := submission.SubmissionID + "/" + string(jobID)
	testS3key := submission.Testset + "/" + submission.TestsetVersion + "/"

	if err := s3m.UploadFileToS3(ctx, srcS3key, body); err != nil {
		return shared.JobSpec{}, err
	}

	jobspec := shared.JobSpec{
		JobId:          string(jobID),
		Language:       submission.Language,
		Version:        submission.Version,
		SubmissionID:   submission.SubmissionID,
		Bucket:         submission.Bucket,
		SrcCodeS3Key:   srcS3key,
		TestsetS3Key:   testS3key,
		Testset:        submission.Testset,
		TestsetVersion: submission.TestsetVersion,
	}

	return jobspec, nil
}
