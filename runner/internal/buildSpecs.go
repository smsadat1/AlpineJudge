// prepares container with all OCI specs
package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"utils"

	oci "github.com/containerd/containerd/oci"
	"github.com/joho/godotenv"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func build_ociSpecOpts(slotID uint32) []oci.SpecOpts {

	godotenv.Load(".env")

	cpuQuota, _ := strconv.ParseFloat(os.Getenv("CPU_QUOTA"), 64)
	memLimitMB, _ := strconv.ParseUint(os.Getenv("MEMORY_LIMIT_MB"), 10, 64)
	pidLimit, _ := strconv.ParseInt(os.Getenv("PID_LIMIT"), 10, 64)
	nnp, _ := strconv.ParseBool(os.Getenv("NO_NEW_PRIVILEGES"))
	rroRootfs, _ := strconv.ParseBool(os.Getenv("READONLY_ROOTFS"))

	fmt.Printf("CQ: %v ML: %v PID: %v NNP: %v RRO: %v", cpuQuota, memLimitMB, pidLimit, nnp, rroRootfs)

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

			/*
				Directory structure
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

func build_agentExecSpec(rules utils.ExecRules) (error, []byte) {

	agentSpec := utils.AgentExecSpec{
		SubmissionID: "",
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
