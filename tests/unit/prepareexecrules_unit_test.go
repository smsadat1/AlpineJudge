package unit_test

import (
	"assert"
	"context"
	"dispatcher"
	"local/runner/executor"
	"local/testrunner/factory"
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

	tf := factory.NewTestFactory(t)
	tf.StartTestMinioS3(t, ctx)

	testSubmissionID := "s234"
	testTestsetID := "ts001"
	testTestsetVer := "v1"
	testSrcCodeS3key := "/submissions/" + testSubmissionID + "/main.cc"
	testTestsetS3key := "/testsets/" + testTestsetID + "/" + testTestsetVer + "/"

	if err := dispatcher.LoadConfigs("../artifacts/runner.config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	if err := utils.LoadRunnerConfigs("../artifacts/runner.config.example.yaml"); err != nil {
		t.Fatal(err)
	}

	data, err := os.Open("../artifacts/main.cpp")
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

	executor.HostSrcFilePath = "/tmp/" + testJobSpec.SubmissionID + "/main.cc"
	executor.HostTestFilePath = "../artifacts/ts001"

	// create temp location
	if err := os.MkdirAll("/tmp/"+testJobSpec.SubmissionID, 0777); err != nil {
		t.Fatalf("Failed creating temp lcocation for test: %v", err)
	}

	err, execrules := executor.PrepareExecrules(ctx, *tf.S3m, testJobSpec, false)
	if err != nil {
		t.Fatal(err)
	}

	expectedImage := "ghcr.io/smsadat1/alpinejudge/gcc:test"
	expectedCompileArgs := []string{
		"g++", "-std=c++20", "-Wall", "-Wextra", "-o", "/tmp/main", "/workspace/main.cc",
	}
	expectedRunArgs := []string{"/tmp/main"}
	expectedCodePathHost := executor.HostSrcFilePath
	expectedCodePathContainer := "/workspace/main.cc"
	expectedTestsetPathHost := executor.HostTestFilePath
	expectedTestsetPathContainer := "/workspace/"

	// Assert using clean struct properties
	assert.String(t, expectedImage, execrules.Image)

	if !slices.Equal(execrules.CompileArgs, expectedCompileArgs) {
		t.Error("Compilation args mismatched")
	}

	if !slices.Equal(execrules.RunArgs, expectedRunArgs) {
		t.Error("Runtime args mismatched")
	}

	assert.String(t, expectedCodePathHost, execrules.CodePathHost)
	assert.String(t, expectedCodePathContainer, execrules.CodePathContainer)
	assert.String(t, expectedTestsetPathContainer, execrules.TestsetPathContainer)
	assert.String(t, expectedTestsetPathHost, execrules.TestsetPathHost)
	assert.Uint64(t, utils.RunCfg.Limits.MemoryLimitMB, execrules.MemoryLimitMB)

	if execrules.CpuQuota != float64(utils.RunCfg.Limits.CPUQuota) {
		t.Errorf("Expected %f, got %f", execrules.CpuQuota, float64(utils.RunCfg.Limits.CPUQuota))
	}

	if execrules.PidLimit != int64(utils.RunCfg.Limits.PIDLimit) {
		t.Errorf("Expected %d, got %d", execrules.PidLimit, int64(utils.RunCfg.Limits.PIDLimit))
	}

	// ajagent.RunnerAgent()
}
