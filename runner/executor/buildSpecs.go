// prepares container with all OCI specs
package executor

import (
	"encoding/json"
	"log"
	"path/filepath"
	"utils"

	oci "github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func Build_ociSpecOpts(rules utils.ExecRules) []oci.SpecOpts {

	memoryBytes := uint64(rules.MemoryLimitMB * 1024 * 1024)
	period := uint64(100000) // 100 ms period
	quota := int64(rules.CpuQuota * float64(period))

	absCodeHost, _ := filepath.Abs(rules.CodePathHost)
	absExecSpecHost, _ := filepath.Abs(rules.ExecutionSpecPathHost)
	absSocketHost, _ := filepath.Abs(rules.HostEventSocket)
	absTestsetHost, _ := filepath.Abs(rules.TestsetPathHost)

	log.Printf("S: %v | T: %v | C: %v | E: %v", absSocketHost, absTestsetHost, absCodeHost, absExecSpecHost)

	opts := []oci.SpecOpts{
		// start with default Linux specs or else OCI spec fails
		oci.WithDefaultSpec(),

		// resource limits
		oci.WithMemoryLimit(memoryBytes),
		// disable memory swap so Linux doesn't give extra memory with it which results to never hitting MLE
		oci.WithMemorySwap(int64(memoryBytes)),
		oci.WithPidsLimit(rules.PidLimit),
		oci.WithCPUCFS(quota, period),

		// mount file
		oci.WithMounts([]specs.Mount{
			// DEBUG only
			// {
			// 	Source:      "/home/pancake/Projects/alpinejudge/runner/ajagent/cmd/ajagent",
			// 	Destination: "/usr/bin/ajagent",
			// 	Type:        "bind",
			// 	Options:     []string{"bind", "ro"},
			// },
			{
				// writable /tmp for temp objects
				Source:      "tmpfs",
				Destination: "/tmp",
				Type:        "tmpfs",
				Options:     []string{"nosuid", "nodev", "mode=1777"},
			},
			{
				Source:      "/tmp/testsub001", // Entire parent folder on host
				Destination: "/workspace",      // Mapped as container root workspace
				Type:        "bind",
				Options:     []string{"rbind", "rw"},
			},
		}),

		oci.WithEnv([]string{
			// Must use this so all necessary tools are available in /usr/bin & /usr/sbin as some images doesn't do that b default
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"CONFIG_PATH=/workspace/execspec.json",
			"TESTSET_PATH=/workspace",
			"STREAM_SOCKET_PATH=/workspace/agent.sock",
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
