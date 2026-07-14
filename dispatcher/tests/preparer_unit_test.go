package tests

import (
	"context"
	"dispatcher"
	"os"
	"testing"
	"time"

	"shared"
)

func Test_SubmissionPreperation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	t.Setenv("TEST_S3_URL", "http://localhost:9000")
	t.Setenv("TEST_S3_USERNAME", "minioadmin")
	t.Setenv("TEST_S3_PASSWORD", "minioadminpassword")
	t.Setenv("TEST_S3_BUCKET_NAME", "ajbucket-test-preparer")
	t.Setenv("TEST_S3_REGION_NAME", "us-east-1")

	if err := dispatcher.LoadConfigs("artifacts/config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	s3m, err := shared.InitS3Manager(
		ctx,
		os.Getenv("TEST_S3_BUCKET_NAME"),
		os.Getenv("TEST_S3_REGION_NAME"),
		os.Getenv("TEST_S3_USERNAME"),
		os.Getenv("TEST_S3_PASSWORD"),
		os.Getenv("TEST_S3_URL"),
	)

	if err != nil {
		t.Error(err)
	}

	_, _ = s3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	if err := s3m.UploadDirToS3(ctx, "ts001/v1", "artifacts/ts001"); err != nil {
		t.Fatal(err)
	}

	submSpec := dispatcher.SubmissionSpec{
		SubmissionID:   "s012",
		Language:       "cpp",
		Version:        "c++17",
		Source:         `#include<stdion.h>\nint main() \n{ std::cout << "Hello World\n";\n}`,
		Testset:        "ts001",
		TestsetVersion: "v1",
	}

	if err := dispatcher.ValidateSubmission(ctx, *s3m, submSpec); err != nil {
		t.Error(err)
	}

	jobSpec, err := dispatcher.PrepareSubmission(ctx, *s3m, submSpec)
	if err != nil {
		t.Error(err)
	}

	if jobSpec.SubmissionID != submSpec.SubmissionID ||
		jobSpec.Language != submSpec.Language ||
		jobSpec.Version != submSpec.Version ||
		jobSpec.SrcCodeS3Key != "submissions/"+submSpec.SubmissionID+"/" ||
		jobSpec.TestsetS3Key != submSpec.Testset+"/"+submSpec.TestsetVersion+"/" ||
		jobSpec.Testset != submSpec.Testset ||
		jobSpec.TestsetVersion != submSpec.TestsetVersion {
		t.Error("Jobspec mismatched or malformed")
	}
}
