package repository

import (
	"dispatcher"
	"shared"
	"testing"
	"utils"
)

type TestRepository struct {
	TestJobSpec   shared.JobSpec
	TestSubmSpec  dispatcher.SubmissionSpec
	TestExecRules utils.ExecRules
}

func NewTestRepository(t *testing.T) *TestRepository {
	t.Helper()

	tjs := shared.JobSpec{
		Language:       "cpp",
		Version:        "c++17",
		SubmissionID:   "testsub001",
		Bucket:         "testbucket",
		SrcCodeS3Key:   "submissions/testsub001/main.cpp",
		TestsetS3Key:   "testsets/ts001/",
		Testset:        "ts001",
		TestsetVersion: "v1",
	}

	tss := dispatcher.SubmissionSpec{
		SubmissionID:   "testsub001",
		Bucket:         "testbucket",
		Language:       "cpp",
		Version:        "c++17",
		Source:         `#include<iostream> int main() {return 0;}`,
		Testset:        "ts001",
		TestsetVersion: "v1",
	}

	ter := utils.ExecRules{
		RunnerID:     "testrunner",
		SubmissionID: "testsub001",
		ContainerID:  "testcontainer",
		Image:        "aplinejudge/gcc:test",
		CompileArgs:  []string{"/usr/bin/g++", "-Wall", "-Wextra", "-o", "../artifacts/main", "../artifacts/main.cpp"},
		RunArgs:      []string{"./../artifacts/main"},
		TestID:       "ts001",

		CodePathHost:         "../artifacts/main.cpp",
		CodePathContainer:    "/workspace/main.cpp",
		TestsetPathHost:      "../aritifacts/ts001",
		TestsetPathContainer: "/workspace/ts001/",
		Env: map[string]string{
			"CONFIG_PATH":        "/workspace/execspec.json",
			"TESTSET_PATH":       "/workspace/ts001/",
			"STREAM_SOCKET_PATH": "/workspace/agentstream.sock",
		},
		EventSocket:    "../artifacts/agent.socket",
		EventQueueName: "test-sse-queue",

		MemoryLimitMB:  256,
		PidLimit:       64,
		CpuQuota:       2.0,
		NoNewPrivilege: true,
		ReadOnlyRootfs: true,
		Timeoutsec:     300,
		LogLimitKB:     512,
	}

	return &TestRepository{
		TestJobSpec:   tjs,
		TestSubmSpec:  tss,
		TestExecRules: ter,
	}
}
