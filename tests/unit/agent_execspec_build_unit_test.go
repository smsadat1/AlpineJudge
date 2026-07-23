package unit_test

import (
	"assert"
	"encoding/json"
	"local/runner/executor"
	"local/testrunner/repository"
	"testing"
	"utils"
)

func Test_Build_agentExecSpec(t *testing.T) {
	tr := repository.NewTestRepository(t)
	testRules := tr.TestExecRules

	err, data := executor.Build_agentExecSpec(testRules)
	if err != nil {
		t.Fatal(err)
	}

	var agentConfig utils.AgentExecSpec
	if err := json.Unmarshal(data, &agentConfig); err != nil {
		t.Fatal(err)
	}

	assert.String(t, testRules.SubmissionID, agentConfig.SubmissionID)
	assert.Uint32(t, testRules.Timeoutsec, agentConfig.TimeoutSec)
	assert.Uint32(t, testRules.LogLimitKB, agentConfig.LogLimitKB)
	assert.String(t, testRules.TestsetPathContainer, agentConfig.TestSetPath)
	assert.Slice(t, agentConfig.CompileArgs, testRules.CompileArgs)
	assert.Slice(t, agentConfig.RunArgs, testRules.RunArgs)
}
