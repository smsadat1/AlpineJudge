// prepares container with all OCI specs
package executor

import (
	"encoding/json"
	"log"
	"utils"

	oci "github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Build_ociSpecOpts(rules utils.ExecRules) []oci.SpecOpts {

	memoryBytes := uint64(rules.MemoryLimitMB * 1024 * 1024)
	period := uint64(100000)

	opts := []oci.SpecOpts{
		// start with default Linux specs or else OCI spec fails
		oci.WithDefaultSpec(),

		// resource limits
		oci.WithMemoryLimit(memoryBytes),
		oci.WithPidsLimit(rules.PidLimit),
		oci.WithCPUCFS(int64(rules.CpuQuota), period),

		// mount file
		oci.WithMounts([]specs.Mount{
			{
				// source code (single file mount)
				Source:      rules.CodePathHost,
				Destination: rules.CodePathContainer,
				Type:        "bind",
				Options:     []string{"bind", "ro"},
			},
			{
				// agent execution specs (single file mount)
				Source:      "/tmp/alpinejudge/" + rules.RunnerID + "/execspec.json",
				Destination: "/workspace/execspec.json",
				Type:        "bind",
				Options:     []string{"bind", "ro"},
			},
			{
				// unix socker for agent to stream execution state
				Source:      rules.EventSocket,
				Destination: "/workspace/agent.sock",
				Type:        "bind",
				Options:     []string{"bind", "rw"},
			},
			{
				// testset (direotory mount)
				Source:      rules.TestsetPathHost,
				Destination: rules.TestsetPathContainer,
				Type:        "bind",
				Options:     []string{"rbind", "ro"},
			},
		}),

		oci.WithEnv([]string{
			"CONFIG_PATH=/workspace/execspec.json",
			"TESTSET_PATH=/workspace/" + rules.TestID + "/",
			"STREAM_SOCKET_PATH=/workspace/agentstream.sock",
		}),
	}

	if rules.NoNewPrivilege {
		opts = append(opts, oci.WithNoNewPrivileges)
	}

	if rules.ReadOnlyRootfs {
		opts = append(opts, oci.WithRootFSReadonly())
	}

	// evaluated last to guarantee execution parameters survive
	opts = append(opts, oci.WithProcessArgs("/usr/bin/ajagent"))

	return opts
}

func Build_agentExecSpec(rules utils.ExecRules) (error, []byte) {

	agentSpec := utils.AgentExecSpec{
		SubmissionID: rules.SubmissionID,
		LogLimitKB:   rules.LogLimitKB,
		TimeoutSec:   rules.Timeoutsec,
		TestSetPath:  "/workspace/" + rules.TestID + "/",
		CompileArgs:  rules.CompileArgs,
		RunArgs:      rules.RunArgs,
	}

	data, err := json.Marshal(agentSpec)
	if err != nil {
		return err, []byte{}
	}

	log.Println("Created agent exespec json")
	return nil, data
}
