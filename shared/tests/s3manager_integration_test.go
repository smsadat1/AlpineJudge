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
	t.Setenv("TEST_S3_BUCKET_NAME", "ajtestbucket1")
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

	s3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	prefix := "testsets/"
	s3key := "submissions/s001/main.py"
	ofile := "artifacts/hudai.py"

	fileBody, err := os.Open("artifacts/main.py")
	if err != nil {
		t.Fatal(err)
	}

	if err := s3m.UploadFileToS3(ctx, s3key, fileBody); err != nil {
		t.Fatal(err)
	}
	if err := s3m.UploadDirToS3(ctx, prefix, "artifacts/ts001"); err != nil {
		t.Fatal(err)
	}

	err = s3m.DownloadFileFromS3(ctx, os.Getenv("TEST_S3_BUCKET_NAME"), s3key, ofile)
	if err != nil {
		t.Fatal(err)
	}

	err = s3m.DownloadDirFromS3(ctx, os.Getenv("TEST_S3_BUCKET_NAME"), prefix, "artifacts/ts002/")
	if err != nil {
		t.Fatal(err)
	}
}
