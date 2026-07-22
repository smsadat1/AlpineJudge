package tests

import (
	"context"
	"dispatcher"
	"local/runner/executor"
	"os"
	"shared"
	"slices"
	"testing"
	"time"
	"utils"
)

func Test_PrepareExecrules(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	t.Setenv("TEST_S3_URL", "http://localhost:9000")
	t.Setenv("TEST_S3_USERNAME", "minioadmin")
	t.Setenv("TEST_S3_PASSWORD", "minioadminpassword")
	t.Setenv("TEST_S3_BUCKET_NAME", "ajbucket-test-preparer")
	t.Setenv("TEST_S3_REGION_NAME", "us-east-1")

	testSubmissionID := "s234"
	testTestsetID := "ts001"
	testTestsetVer := "v1"
	testSrcCodeS3key := "/submissions/" + testSubmissionID + "/main.cc"
	testTestsetS3key := "/testsets/" + testTestsetID + "/" + testTestsetVer + "/"

	if err := dispatcher.LoadConfigs("../artifacts/config.example.yaml"); err != nil {
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

	if err := utils.LoadRunnerConfigs("../artifacts/config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	data, err := os.Open("artifacts/main.cc")
	if err != nil {
		t.Fatal(err)
	}

	// upload artifacts first for test
	s3m.UploadFileToS3(ctx, testSrcCodeS3key, data)
	s3m.UploadDirToS3(ctx, testTestsetS3key, "../artifacts/ts001")

	_, _ = s3m.CreateABucket(ctx, os.Getenv("TEST_S3_BUCKET_NAME"))

	testJobSpec := shared.JobSpec{
		Language:       "cc",
		Version:        "c++20",
		SubmissionID:   testSubmissionID,
		Bucket:         os.Getenv("TEST_S3_BUCKET_NAME"),
		SrcCodeS3Key:   testSrcCodeS3key,
		TestsetS3Key:   testTestsetS3key,
		Testset:        testTestsetID,
		TestsetVersion: testTestsetVer,
	}

	executor.HostSrcFilePath = "artifacts/main.cc"
	executor.HostTestFilePath = "artifacts/ts001"
	err, execrules := executor.PrepareExecrules(ctx, *s3m, testJobSpec, true)
	if err != nil {
		t.Fatal(err)
	}

	expectedImage := "ghcr.io/smsadat1/ajgcc:v0.1.0"
	expectedCompileArgs := []string{
		"/usr/bin/g++", "-std=c++20", "-Wall", "-Wextra", "-o", "main", "main.cc",
	}
	expectedRunArgs := []string{"./main"}
	expectedCodePathHost := executor.HostSrcFilePath
	expectedCodePathContainer := "/workspace/main.cc"
	expectedTestsetPathHost := executor.HostTestFilePath
	expectedTestsetPathContainer := "/workspace/" + testTestsetID + "/" + testTestsetVer + "/"

	// Assert using clean struct properties
	if execrules.Image != expectedImage {
		t.Errorf("Expected %s, got %s", execrules.Image, expectedImage)
	}

	if !slices.Equal(execrules.CompileArgs, expectedCompileArgs) {
		t.Error("Compilation args mismatched")
	}

	if !slices.Equal(execrules.RunArgs, expectedRunArgs) {
		t.Error("Runtime args mismatched")
	}

	if execrules.CodePathHost != expectedCodePathHost {
		t.Errorf("Expected %s, got %s", execrules.CodePathHost, expectedCodePathHost)
	}

	if execrules.CodePathContainer != expectedCodePathContainer {
		t.Errorf("Expected %s, got %s", execrules.CodePathContainer, expectedCodePathContainer)
	}

	if execrules.TestsetPathContainer != expectedTestsetPathContainer {
		t.Errorf("Expected %s, got %s", execrules.TestsetPathContainer, expectedTestsetPathContainer)
	}

	if execrules.TestsetPathHost != expectedTestsetPathHost {
		t.Errorf("Expected %s, got %s", execrules.TestsetPathHost, expectedTestsetPathHost)
	}

	if execrules.MemoryLimitMB != utils.RunCfg.Limits.MemoryLimitMB {
		t.Errorf("Expected %d, got %d", execrules.MemoryLimitMB, utils.RunCfg.Limits.MemoryLimitMB)
	}

	if execrules.CpuQuota != float64(utils.RunCfg.Limits.CPUQuota) {
		t.Errorf("Expected %f, got %f", execrules.CpuQuota, float64(utils.RunCfg.Limits.CPUQuota))
	}

	if execrules.PidLimit != int64(utils.RunCfg.Limits.PIDLimit) {
		t.Errorf("Expected %d, got %d", execrules.PidLimit, int64(utils.RunCfg.Limits.PIDLimit))
	}

}
