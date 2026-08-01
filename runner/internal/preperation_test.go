package internal

import (
	"assert"
	"context"
	"local/testrunner/factory"
	"os"
	"shared"
	"testing"
	"time"
	"utils"
)

func Test_PrepareExecrules(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tf := factory.NewTestFactory(t)
	tf.StartTestMinioS3(t, ctx)

	testSubmissionID := "s234"
	testTestsetID := "ts001"
	testTestsetVer := "v1"
	testSrcCodeS3key := "/submissions/" + testSubmissionID + "/main.cc"
	testTestsetS3key := "/testsets/" + testTestsetID + "/" + testTestsetVer + "/"

	data, err := os.Open("../examples/main.cpp")
	if err != nil {
		t.Fatal(err)
	}

	// upload artifacts first for test
	tf.S3m.UploadFileToS3(ctx, testSrcCodeS3key, data)
	tf.S3m.UploadDirToS3(ctx, testTestsetS3key, "../artifacts/ts001")

	_, _ = tf.S3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	testJobSpec := shared.JobSpec{
		Language:       "cc",
		Version:        "c++20",
		SubmissionID:   testSubmissionID,
		Bucket:         os.Getenv("TEST_S3_BUCKET_NAME"),
		SrcCodeS3Key:   testSrcCodeS3key,
		TestsetS3Key:   testTestsetS3key,
		Testset:        testTestsetID,
		TestsetVersion: testTestsetVer,
		Image:          "ghcr.io/smsadat1/alpinejudge/gcc:test",
	}

	// create temp location
	if err := os.MkdirAll("/tmp/"+testJobSpec.SubmissionID, 0777); err != nil {
		t.Fatalf("Failed creating temp lcocation for test: %v", err)
	}

	err, execrules := prepareExecrules(ctx, *tf.S3m, testJobSpec)
	if err != nil {
		t.Fatal(err)
	}

	submmittedCodePathBase := "/workspace/submissions/" + testJobSpec.SubmissionID + "/"
	expectedImage := "ghcr.io/smsadat1/alpinejudge/gcc:test"
	expectedCompileArgs := []string{
		"g++", "-std=c++20", "-Wall", "-Wextra", "-o", "/tmp/main", submmittedCodePathBase + "main.cc",
	}
	expectedRunArgs := []string{"/tmp/main"}

	assert.String(t, expectedImage, execrules.Image)
	assert.Slice(t, execrules.CompileArgs, expectedCompileArgs)
	assert.Slice(t, execrules.RunArgs, expectedRunArgs)
	assert.Uint64(t, 0, execrules.MemoryLimitMB)

	if execrules.CpuQuota != float64(utils.RunCfg.Limits.CPUQuota) {
		t.Errorf("Expected %f, got %f", execrules.CpuQuota, float64(utils.RunCfg.Limits.CPUQuota))
	}

	if execrules.PidLimit != int64(utils.RunCfg.Limits.PIDLimit) {
		t.Errorf("Expected %d, got %d", execrules.PidLimit, int64(utils.RunCfg.Limits.PIDLimit))
	}
}
