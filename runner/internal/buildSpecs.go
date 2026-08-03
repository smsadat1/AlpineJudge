// prepares container with all OCI specs
package internal

import (
	"fmt"
	"os"
	"strconv"
	"utils"

	oci "github.com/containerd/containerd/oci"
	"github.com/joho/godotenv"
	"github.com/opencontainers/runtime-spec/specs-go"
)

/*
	Directory structure of the host temp location for runner
	/tmp/runner/
		sockets/
			1.sock
			2.sock
			3.sock
			...
		testsets/
		 	ts001/
			ts002/
			ts003/
			...
		submissions/
			s001/
			s002/
			s003/
			...
*/

// Collects parameters from env vars and set to container's Specs
func build_ociSpecOpts(slotID uint32) []oci.SpecOpts {

	godotenv.Load(".env")

	cpuQuota, _ := strconv.ParseFloat(os.Getenv("CPU_QUOTA"), 64)
	memLimitMB, _ := strconv.ParseUint(os.Getenv("MEMORY_LIMIT_MB"), 10, 64)
	pidLimit, _ := strconv.ParseInt(os.Getenv("PID_LIMIT"), 10, 64)
	nnp, _ := strconv.ParseBool(os.Getenv("NO_NEW_PRIVILEGES"))
	rroRootfs, _ := strconv.ParseBool(os.Getenv("READONLY_ROOTFS"))

	fmt.Printf("CQ: %v ML: %v PID: %v NNP: %v RRO: %v\n", cpuQuota, memLimitMB, pidLimit, nnp, rroRootfs)

	memoryBytes := uint64(memLimitMB * 1024 * 1024)
	period := uint64(100000) // 100 ms period
	quota := int64(cpuQuota * float64(period))

	opts := []oci.SpecOpts{
		// start with default Linux specs or else OCI spec fails
		oci.WithDefaultSpec(),

		// resource limits
		oci.WithMemoryLimit(memoryBytes),
		// fixed memory swap so Linux doesn't abuse swap to give extra memory without limits
		oci.WithMemorySwap(int64(memoryBytes)),

		// this function isn't available (use spec.Proccess.OOMScoreAdj instead)
		// oci.WithOOMScoreAdj(888) // 888 value set container processes in high-priority for OOM kills so container gets terminated early in case of memory abuse

		oci.WithPidsLimit(pidLimit),
		oci.WithCPUCFS(quota, period),

		// mount file
		oci.WithMounts([]specs.Mount{
			{
				// writable /tmp for temp objects
				Source:      "tmpfs",
				Destination: "/tmp",
				Type:        "tmpfs",
				Options:     []string{"nosuid", "nodev", "mode=1777"},
			},
			{
				Source:      "/tmp/runner/", // Entire parent folder on host
				Destination: "/workspace",   // Mapped as container root workspace
				Type:        "bind",
				Options:     []string{"rbind", "rw"},
			},
		}),

		oci.WithEnv([]string{
			// Must use this so all necessary tools are available in /usr/bin & /usr/sbin as some images doesn't do that b default
			"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			"STREAM_SOCKET_PATH=/workspace/sockets/" + fmt.Sprint(slotID) + ".sock",
		}),
	}

	if nnp {
		opts = append(opts, oci.WithNoNewPrivileges)
	}

	if rroRootfs {
		opts = append(opts, oci.WithRootFSReadonly())
	}

	// evaluated last to guarantee execution parameters survive
	opts = append(opts, oci.WithProcessArgs("/usr/bin/ajagent"))

	return opts
}

func Build_AgentExecSpec(rules utils.ExecRules) utils.AgentExecSpec {

	agentSpec := utils.AgentExecSpec{
		SubmissionID:  "",
		LogLimitKB:    rules.PerTestLogLimitKB,
		TimeoutSec:    rules.PerTestTimeoutsec,
		MemoryLimitMB: rules.PerTestMemoryLimitMB,
		TestSetPath:   "/workspace/testsets/" + rules.TestID + "/",
		CompileArgs:   rules.CompileArgs,
		RunArgs:       rules.RunArgs,
	}

	return agentSpec
}
