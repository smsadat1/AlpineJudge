package repository

import (
	"dispatcher"
	"path/filepath"
	"shared"
	"testing"
	"utils"

	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type TestRepository struct {
	TestJobSpec     shared.JobSpec
	TestSubmSpec    dispatcher.SubmissionSpec
	TestExecRules   utils.ExecRules
	TestOCISpecOpts []oci.SpecOpts
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
		CompileArgs:  []string{"/usr/local/bin/g++", "-Wall", "-Wextra", "-o", "/tmp/main", "/workspace/main.cpp"},
		RunArgs:      []string{"/tmp/main"},
		TestID:       "ts001",

		// CodePathHost:               "../artifacts/main.cpp",
		CodePathContainer:          "/workspace/main.cpp",
		TestsetPathHost:            "../artifacts/ts001",
		TestsetPathContainer:       "/workspace/ts001/",
		ExecutionSpecPathHost:      "../artifacts/execspec1.json",
		ExecutionSpecPathContainer: "/workspace/execspec.json",
		HostEventSocket:            "../artifacts/agent.sock",
		ContainerEventSocket:       "/workspace/agent.sock",
		EventQueueName:             "test-sse-queue",

		Env: map[string]string{
			"CONFIG_PATH":        "/workspace/execspec.json",
			"TESTSET_PATH":       "/workspace/ts001/",
			"STREAM_SOCKET_PATH": "/workspace/agent.sock",
		},

		MemoryLimitMB:  1024,
		PidLimit:       64,
		CpuQuota:       2.0,
		NoNewPrivilege: true,
		ReadOnlyRootfs: true,
		Timeoutsec:     25,
		LogLimitKB:     512,
	}

	return &TestRepository{
		TestJobSpec:   tjs,
		TestSubmSpec:  tss,
		TestExecRules: ter,
	}
}

func (tr *TestRepository) NewTestOCISpecOpts(t *testing.T, testRules utils.ExecRules) {

	t.Helper()

	memoryBytes := uint64(testRules.MemoryLimitMB * 1024 * 1024)
	period := uint64(100000) // 100 ms period
	quota := int64(testRules.CpuQuota * float64(period))

	absCodeHost, _ := filepath.Abs(testRules.CodePathHost)
	absExecSpecHost, _ := filepath.Abs(testRules.ExecutionSpecPathHost)
	absSocketHost, _ := filepath.Abs(testRules.HostEventSocket)
	absTestsetHost, _ := filepath.Abs(testRules.TestsetPathHost)

	opts := []oci.SpecOpts{
		// start with default Linux specs or else OCI spec fails
		oci.WithDefaultSpec(),

		// resource limits
		oci.WithMemoryLimit(memoryBytes),
		// disable memory swap so Linux doesn't give extra memory with it which results to never hitting MLE
		oci.WithMemorySwap(int64(memoryBytes)),
		oci.WithPidsLimit(testRules.PidLimit),
		oci.WithCPUCFS(quota, period),

		// mount file
		oci.WithMounts([]specs.Mount{

			// DEBUG only
			{
				Source:      "/home/pancake/Projects/alpinejudge/runner/ajagent/cmd/ajagent",
				Destination: "/usr/bin/ajagent",
				Type:        "bind",
				Options:     []string{"bind", "ro"},
			},

			{
				// writable /tmp for temp objects
				Source:      "tmpfs",
				Destination: "/tmp",
				Type:        "tmpfs",
				Options:     []string{"nosuid", "nodev", "mode=1777"},
			},
			{
				// source code (single file mount)
				Source:      absCodeHost,
				Destination: testRules.CodePathContainer,
				Type:        "bind",
				Options:     []string{"bind", "ro"},
			},
			{
				// agent execution specs (single file mount)
				Source:      absExecSpecHost,
				Destination: testRules.ExecutionSpecPathContainer,
				Type:        "bind",
				Options:     []string{"bind", "ro"},
			},
			{
				// unix socker for agent to stream execution state
				Source:      absSocketHost,
				Destination: testRules.ContainerEventSocket,
				Type:        "bind",
				Options:     []string{"bind", "rw"},
			},
			{
				// testset (direotory mount)
				Source:      absTestsetHost,
				Destination: testRules.TestsetPathContainer,
				Type:        "bind",
				Options:     []string{"rbind", "ro"},
			},
		}),

		oci.WithEnv([]string{
			// Must use this so all necessary tools are available in /usr/bin & /usr/sbin as some images doesn't do that b default
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"CONFIG_PATH=/workspace/execspec.json",
			"TESTSET_PATH=/workspace/" + testRules.TestID + "/",
			"STREAM_SOCKET_PATH=/workspace/agent.sock",
		}),
	}

	if testRules.NoNewPrivilege {
		opts = append(opts, oci.WithNoNewPrivileges)
	}

	if testRules.ReadOnlyRootfs {
		opts = append(opts, oci.WithRootFSReadonly())
	}

	// evaluated last to guarantee execution parameters survive
	opts = append(opts, oci.WithProcessArgs("/usr/bin/ajagent"))

	tr.TestOCISpecOpts = opts
}
