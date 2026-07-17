package tests

import (
	"encoding/json"
	"local/runner/executor"
	"local/runner/utils"
	"slices"
	"testing"
)

func Test_Build_agentExecSpec(t *testing.T) {

	var testEnv map[string]string
	testEnv = make(map[string]string)
	testEnv["CONFIG_PATH"] = "/workspace/execspec.json"

	testRules := utils.ExecRules{
		RunnerID:    "runner-001",
		ContainerID: "container-123",
		Image:       "ghcr.io/smsadat1/ajgo:v0.1.0",
		CompileArgs: []string{"/usr/local/go1.26/bin/go"},
		RunArgs:     []string{"/usr/local/go1.26/bin/go", "run", "main.go"},
		TestID:      "ts123",

		CodePathHost:         "/tmp/alpinejudge/runner-001/main.go",
		CodePathContainer:    "/workspace/main.go",
		TestsetPathHost:      "/tmp/alpinejudge/runner-001/" + "ts123/",
		TestsetPathContainer: "/workspace/" + "ts123/",
		Env:                  testEnv,
		OutStreamQueueName:   "IdkWhatToName",
		ErrStreamQueueName:   "IdkWhatToName",

		MemoryLimitMB:  1024,
		CpuQuota:       2,
		PidLimit:       128,
		NoNewPrivilege: true,
		ReadOnlyRootfs: true,
		LogLimitKB:     234,
		Timeoutsec:     300,
	}

	err, data := executor.Build_agentExecSpec(testRules)
	if err != nil {
		t.Fatal(err)
	}

	var agentConfig utils.AgentExecSpec
	if err := json.Unmarshal(data, &agentConfig); err != nil {
		t.Fatal(err)
	}

	// Assert using clean struct properties
	if agentConfig.RunnerID != testRules.RunnerID {
		t.Errorf("Expected %s, got %s", testRules.RunnerID, agentConfig.RunnerID)
	}

	if agentConfig.TimeoutSec != testRules.Timeoutsec {
		t.Errorf("Expected %d, got %d", testRules.Timeoutsec, agentConfig.TimeoutSec)
	}

	if agentConfig.LogLimitKB != testRules.LogLimitKB {
		t.Errorf("Expected %d, got %d", testRules.LogLimitKB, agentConfig.LogLimitKB)
	}

	if agentConfig.TestSetPath != testRules.TestsetPathContainer {
		t.Errorf("Expected %s, got %s", testRules.TestsetPathContainer, agentConfig.TestSetPath)
	}

	if !slices.Equal(agentConfig.CompileArgs, testRules.CompileArgs) {
		t.Errorf("Compilation args mismatched\n")
	}

	if !slices.Equal(agentConfig.RunArgs, testRules.RunArgs) {
		t.Errorf("Runtime args mismatched\n")
	}
}
