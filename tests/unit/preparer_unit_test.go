package unit_test

import (
	"context"
	"dispatcher"
	"local/testrunner/factory"
	"os"
	"testing"
	"time"
)

func Test_SubmissionPreperation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tf := factory.NewTestFactory(t)
	tf.StartTestMinioS3(t, ctx)

	if err := dispatcher.LoadConfigs("../artifacts/config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	_, _ = tf.S3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	if err := tf.S3m.UploadDirToS3(ctx, "ts001/v1", "../artifacts/ts001"); err != nil {
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

	if err := dispatcher.ValidateSubmission(ctx, *tf.S3m, submSpec); err != nil {
		t.Error(err)
	}

	jobSpec, err := dispatcher.PrepareSubmission(ctx, *tf.S3m, submSpec)
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
