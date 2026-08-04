package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"shared"
	"testing"
	"utils"

	"github.com/containerd/containerd/oci"
)

type RunnerTestRepository struct {
	TestSubmissionID  string
	TestsetID         string
	TestJobSpec       shared.JobSpec
	TestExecRules     utils.ExecRules
	TestOCISpecOpts   []oci.SpecOpts
	TestAgentExecSpec utils.AgentExecSpec
}

func NewRunnerTestRepository(t *testing.T) *RunnerTestRepository {
	t.Helper()

	tsID := "test001"
	testsetID := "ts001"

	// testsetDir := fmt.Sprintf("/tmp/runner/testsets/%s", testsetID)
	// submissionDir := fmt.Sprintf("/tmp/runner/submissions/%s", tsID)

	// Wipe stale leftovers from previous runs FIRST
	// _ = os.RemoveAll(testsetDir)
	// _ = os.RemoveAll(submissionDir)

	// // create neccessary temp locations
	// fmt.Println("\nCreating necessary temp locations")
	// dirs := []string{
	// 	"/tmp/runner/sockets",
	// 	"/tmp/runner/testsets/" + testsetID + "/",
	// 	"/tmp/runner/submissions/" + tsID,
	// }

	// for _, dir := range dirs {
	// 	if err := os.MkdirAll(dir, 0777); err != nil {
	// 		t.Errorf("failed to create directory %s: %v", dir, err)
	// 	}
	// }

	tjs := shared.JobSpec{
		Language: "cpp",
		Version:  "c++17",
		Image:    "ghcr.io/smsadat1/alpinejudge/master:test",
		Source: `
		#include <iostream>

		int main() {
			
		    int a, b = 0;
		    std::cin >> a >> b;
		    std::cout << a + b;
			
		    return 0;   
		}`,
		SubmissionID:         tsID,
		Bucket:               "testbucket",
		SrcCodeS3Key:         fmt.Sprintf("submissions/%s/main.cpp", tsID),
		TestsetS3Key:         fmt.Sprintf("testsets/%s/main.cpp", tsID),
		Testset:              testsetID,
		TestsetVersion:       "v1",
		SSEQueue:             "queue-001",
		PerTestMemoryLimitMB: 1024,
		PerTestLogLimitKB:    512,
		PerTestTimeoutsec:    5,
	}

	ter := utils.ExecRules{
		RunnerID:     "testrunner",
		SubmissionID: "testsub001",
		ContainerID:  "testcontainer",
		Image:        "aplinejudge/gcc:test",

		TestID: "ts001",

		EventQueueName: "test-sse-queue",

		PerTestMemoryLimitMB: 1024,
		PerTestTimeoutsec:    25,
		PerTestLogLimitKB:    512,
	}

	tas := utils.AgentExecSpec{
		SubmissionID:     "testsub099",
		HaltOnFirstError: false,
		LogLimitKB:       512,
		TimeoutSec:       10,
		MemoryLimitMB:    1024,
		TestSetPath:      fmt.Sprintf("/workspace/testsets/%v/", testsetID),
		CompileArgs:      []string{"g++", "-Wall", "-Wextra", "-o", "/tmp/main", fmt.Sprintf("/workspace/submissions/%v/main.cpp", tsID)},
		RunArgs:          []string{"/tmp/main"},
	}

	return &RunnerTestRepository{
		TestSubmissionID:  tsID,
		TestsetID:         testsetID,
		TestJobSpec:       tjs,
		TestExecRules:     ter,
		TestAgentExecSpec: tas,
	}
}

func (rtr *RunnerTestRepository) CreateTempLocations(t *testing.T) {

	t.Helper()

	testsetDir := fmt.Sprintf("/tmp/runner/testsets/%s", rtr.TestsetID)
	submissionDir := fmt.Sprintf("/tmp/runner/submissions/%s", rtr.TestSubmissionID)

	// Wipe stale leftovers from previous runs FIRST
	_ = os.RemoveAll(testsetDir)
	_ = os.RemoveAll(submissionDir)

	// create neccessary temp locations
	fmt.Println("\nCreating necessary temp locations")
	dirs := []string{
		"/tmp/runner/sockets",
		"/tmp/runner/testsets/" + rtr.TestsetID + "/",
		"/tmp/runner/submissions/" + rtr.TestSubmissionID,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			t.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
}

func (rtr *RunnerTestRepository) CopyFiles(t *testing.T, codeFilePath string) {
	t.Helper()

	// copy test artifacts files
	scrData, err := os.ReadFile(codeFilePath)
	if err != nil {
		t.Errorf("Failed copying source file (read): %v", err)
	}

	err = os.WriteFile(fmt.Sprintf("/tmp/runner/submissions/%s/main.cpp", rtr.TestSubmissionID), scrData, 0777)
	if err != nil {
		t.Errorf("Failed copying source file (write): %v", err)
	}

	testsetPath := "../examples/ts001"

	for i := 1; ; i++ {
		input := filepath.Join(testsetPath, fmt.Sprintf("%03din.txt", i))
		output := filepath.Join(testsetPath, fmt.Sprintf("%03dout.txt", i))

		inData, err := os.ReadFile(input)
		if os.IsNotExist(err) {
			break // Normal exit when no more testcases exist
		}

		inData, err = os.ReadFile(input)
		if err != nil {
			t.Errorf("Failed copying source file (read .in): %v", err)
		}

		err = os.WriteFile(fmt.Sprintf("/tmp/runner/testsets/%s/%03din.txt", rtr.TestsetID, i), inData, 0777)
		if err != nil {
			t.Errorf("Error copying testset files (write .in): %v", err)
		}

		outData, err := os.ReadFile(output)
		if err != nil {
			t.Errorf("Failed copying source file (read .out): %v", err)
		}

		err = os.WriteFile(fmt.Sprintf("/tmp/runner/testsets/%s/%03dout.txt", rtr.TestsetID, i), outData, 0777)
		if err != nil {
			t.Errorf("Error copying testset files (write .out): %v", err)
		}
	}

}
