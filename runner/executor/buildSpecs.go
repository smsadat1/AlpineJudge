// prepares container with all OCI specs
package executor

import (
	"local/runner/utils"

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
				Source:      rules.CodePathContainer,
				Destination: rules.CodePathHost,
				Type:        "bind",
				// single file mounth
				Options: []string{"bind", "ro"},
			},
			{
				Source:      rules.TestsetPathHost,
				Destination: rules.TestsetPathContainer,
				Type:        "bind",
				// directory mount
				Options: []string{"rbind", "ro"},
			},
		}),
	}

	if rules.ReadOnlyRootfs {
		opts = append(opts, oci.WithRootFSReadonly())
	}

	// evaluated last to guarantee execution parameters survive
	opts = append(opts, oci.WithProcessArgs(rules.Args...))

	return opts
}
