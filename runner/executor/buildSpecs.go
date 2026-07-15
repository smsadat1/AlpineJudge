// prepares container with all OCI specs
package executor

import (
	"encoding/json"
	"fmt"
	"local/runner/utils"
	"log"
	"os"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func build_ociSpecOpts(
	image containerd.Image, rules utils.ExecRules,
) []oci.SpecOpts {

	memoryBytes := uint64(rules.MemoryLimitMB * 1024 * 1024)
	period := uint64(100000)

	opts := []oci.SpecOpts{
		// image
		oci.WithImageConfig(image),

		// resource limits
		oci.WithMemoryLimit(memoryBytes),
		oci.WithPidsLimit(rules.PidLimit),
		oci.WithCPUCFS(int64(rules.CpuQuota), period),

		// mount file
		oci.WithMounts([]specs.Mount{
			{
				// source code (single file mount)
				Source:      rules.CodePathContainer,
				Destination: rules.CodePathHost,
				Type:        "bind",
				Options:     []string{"bind", "ro"},
			},
			{
				// agent execution specs (single file mount)
				Source:      "/tmp/execspec.json",
				Destination: "/workspace/execspec.json",
				Type:        "bind",
				Options:     []string{"bind", "ro"},
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

func build_agentExecSpec(rules utils.ExecRules) error {

	agentSpec := utils.AgentExecSpec{
		LogLimitKB:  rules.LogLimitKB,
		TimeoutSec:  rules.Timeoutsec,
		TestSetPath: rules.TestID,
		CompileArgs: rules.CompileArgs,
		RunArgs:     rules.RunArgs,
	}

	data, err := json.Marshal(agentSpec)
	if err != nil {
		return err
	}

	if err := os.WriteFile("/tmp/execspec.json", data, os.ModeAppend); err != nil {
		return fmt.Errorf("Failed to create agent execspec json:  %v\n", err)
	}
	log.Println("Created agent exespec json")
	return nil
}
