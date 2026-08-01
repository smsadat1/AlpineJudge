package internal

import (
	"assert"
	"context"
	"testing"
	"time"
	"utils"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/v2/pkg/namespaces"
)

func Test_Build_ociSpecOpts(t *testing.T) {

	var testEnv map[string]string
	testEnv = make(map[string]string)
	testEnv["CONFIG_PATH"] = "/workspace/execspec.json"

	testRules := utils.ExecRules{
		CompileArgs: []string{""},
		RunArgs:     []string{""},
		TestID:      "ts123",

		EventQueueName: "sse-event-queue",

		Env: testEnv,

		MemoryLimitMB:  1024,
		CpuQuota:       2,
		PidLimit:       128,
		NoNewPrivilege: true,
		ReadOnlyRootfs: true,
		LogLimitKB:     234,
		Timeoutsec:     300,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	ctx = namespaces.WithNamespace(ctx, "test_build_oci_spec")
	defer cancel()

	var dummyContainer containers.Container
	var dummyClient oci.Client

	testSlotID := 420
	testOpts := build_ociSpecOpts(uint32(testSlotID))

	testOCISpecs, err := oci.GenerateSpec(ctx, dummyClient, &dummyContainer, testOpts...)
	if err != nil {
		t.Fatal(err)
	}

	assert.Bool(t, testRules.NoNewPrivilege, testOCISpecs.Process.NoNewPrivileges)
	assert.Bool(t, testRules.ReadOnlyRootfs, testOCISpecs.Root.Readonly)
	assert.Slice(t, testOCISpecs.Process.Args, []string{"/usr/bin/ajagent"})

	// Linux Kernel Resource Constraints (cgroups)
	if testOCISpecs.Linux != nil && testOCISpecs.Linux.Resources != nil {
		res := testOCISpecs.Linux.Resources

		// Memory Limit (OCI expects bytes: MB * 1024 * 1024)
		expectedMemoryBytes := int64(testRules.MemoryLimitMB * 1024 * 1024)
		assert.Int64(t, expectedMemoryBytes, *res.Memory.Limit)

		// PIDs Limit
		expectedPidLimit := int64(testRules.PidLimit)
		assert.Int64(t, expectedPidLimit, res.Pids.Limit)

		// default cgroup period is usually 100000 microseconds
		expectedQuota := int64(testRules.CpuQuota) * 100000
		// warn: values passed here are int64, risks of loosing precision in future
		assert.Int64(t, expectedQuota, *res.CPU.Quota)
	} else {
		t.Error("Expected Linux Resources section to be defined for cgroup validation")
	}

	// TODO: Implement storage bind mounts test
}
