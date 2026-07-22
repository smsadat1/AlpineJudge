package unit_test

import (
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
	}

	executor.HostSrcFilePath = "../artifacts/main.cc"
	executor.HostTestFilePath = "../artifacts/ts001"
	err, execrules := executor.PrepareExecrules(ctx, *tf.S3m, testJobSpec, true)
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

	// ajagent.RunnerAgent()
}
