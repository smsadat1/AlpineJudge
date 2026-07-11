package tests

import (
	"context"
	"os"
	"shared"
	"testing"
	"time"
)

func TestS3Manager(t *testing.T) {

	t.Setenv("TEST_S3_URL", "http://localhost:9000")
	t.Setenv("TEST_S3_USERNAME", "minioadmin")
	t.Setenv("TEST_S3_PASSWORD", "minioadminpassword")
	t.Setenv("TEST_S3_BUCKET_NAME", "ajbucket")
	t.Setenv("TEST_S3_REGION_NAME", "us-east-1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s3m, err := shared.InitS3Manager(
		ctx,
		os.Getenv("TEST_S3_BUCKET_NAME"),
		os.Getenv("TEST_S3_REGION_NAME"),
		os.Getenv("TEST_S3_USERNAME"),
		os.Getenv("TEST_S3_PASSWORD"),
		os.Getenv("TEST_S3_URL"),
	)

	if err != nil {
		t.Fatal(err)
	}

	s3key := "submission/s001/main.py"
	fileBody, err := os.Open("hudai.py")
	if err != nil {
		t.Fatal(err)
	}
	s3m.UploadToS3(ctx, s3key, fileBody)

}
