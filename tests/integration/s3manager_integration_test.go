package integration

import (
	"context"
	"local/testrunner/factory"
	"os"
	"testing"
	"time"
)

func TestS3Manager(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tf := factory.NewTestFactory(t)
	tf.StartTestMinioS3(t, ctx)

	prefix := "testsets/"
	s3key := "submissions/s001/main.py"
	ofile := "../artifacts/hudai.py"

	fileBody, err := os.Open("../artifacts/main.py")
	if err != nil {
		t.Fatal(err)
	}

	if err := tf.S3m.UploadFileToS3(ctx, s3key, fileBody); err != nil {
		t.Fatal(err)
	}
	if err := tf.S3m.UploadDirToS3(ctx, prefix, "../artifacts/ts001"); err != nil {
		t.Fatal(err)
	}

	err = tf.S3m.DownloadFileFromS3(ctx, os.Getenv("TEST_S3_BUCKET_NAME"), s3key, ofile)
	if err != nil {
		t.Fatal(err)
	}

	err = tf.S3m.DownloadDirFromS3(ctx, os.Getenv("TEST_S3_BUCKET_NAME"), prefix, "../artifacts/ts002/")
	if err != nil {
		t.Fatal(err)
	}
}
