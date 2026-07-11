package tests

import (
	"context"
	"dispatcher"
	"testing"
	"time"

	"shared"
)

func Test_SubmissionPreperation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	s3m, err := shared.InitS3Manager(ctx,
		"aj-bucket",
		"us-east-1",          // useful for AWS
		"minioadmin",         // Root User
		"minioadminpassword", // Root Password
		"http://localhost:9000",
	)

	if err != nil {
		t.Error(err)
	}

	submSpec := dispatcher.SubmissionSpec{
		SubmissionID:   "s001",
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

	if jobSpec.JobId == "" ||
		jobSpec.SubmissionID != submSpec.SubmissionID ||
		jobSpec.Language != submSpec.Language ||
		jobSpec.Version != submSpec.Language ||
		jobSpec.S3Key != submSpec.SubmissionID+"/"+string(jobSpec.JobId) ||
		jobSpec.Testset != submSpec.Testset ||
		jobSpec.TestsetVersion != submSpec.TestsetVersion {
		t.Error("Jobspec mismatched or malformed")
	}
}
