package unit_test

import (
	"context"
	"local/runner/executor"
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
		RunnerID:    "runner-001",
		ContainerID: "container-123",
		Image:       "ghcr.io/smsadat1/ajgo:v0.1.0",
		CompileArgs: []string{""},
		RunArgs:     []string{""},
		TestID:      "ts123",

		CodePathHost:         "/tmp/alpinejudge/runner-001/main.go",
		CodePathContainer:    "/workspace/main.go",
		TestsetPathHost:      "/tmp/alpinejudge/runner-001/" + "ts123/",
		TestsetPathContainer: "/workspace/" + "ts123/",
		Env:                  testEnv,
		EventSocket:          "../artifacts/agent.sock",

		MemoryLimitMB:  1024,
		CpuQuota:       2,
		PidLimit:       128,
		NoNewPrivilege: true,
		ReadOnlyRootfs: true,
		LogLimitKB:     234,
		Timeoutsec:     300,
	}

	testOpts := executor.Build_ociSpecOpts(testRules)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	ctx = namespaces.WithNamespace(ctx, "test_build_oci_spec")
	defer cancel()

	var dummyContainer containers.Container
	var dummyClient oci.Client

	testOCISpecs, err := oci.GenerateSpec(ctx, dummyClient, &dummyContainer, testOpts...)
	if err != nil {
		t.Fatal(err)
	}

	// ASSERTION 1: Security Privileges
	if testOCISpecs.Process.NoNewPrivileges != testRules.NoNewPrivilege {
		t.Errorf("Expected NoNewPrivileges to be %t, got %t", testRules.NoNewPrivilege, testOCISpecs.Process.NoNewPrivileges)
	}

	// ASSERTION 2: Process Namespaces / Host Setup (Optional check depending on your executor)
	// if testOCISpecs.Process.Args == nil {
	// 	t.Error("Expected container entrypoint process args to be initialized, got nil")
	// }

	// ASSERTION 3: Read-Only Root Filesystem
	if testOCISpecs.Root != nil && testOCISpecs.Root.Readonly != testRules.ReadOnlyRootfs {
		t.Errorf("Expected Root Readonly to be %t, got %t", testRules.ReadOnlyRootfs, testOCISpecs.Root.Readonly)
	}

	// ASSERTION 4: Linux Kernel Resource Constraints (cgroups)
	if testOCISpecs.Linux != nil && testOCISpecs.Linux.Resources != nil {
		res := testOCISpecs.Linux.Resources

		// Memory Limit (OCI expects bytes: MB * 1024 * 1024)
		expectedMemoryBytes := int64(testRules.MemoryLimitMB * 1024 * 1024)
		if res.Memory != nil && *res.Memory.Limit != expectedMemoryBytes {
			t.Errorf("Expected memory limit %d bytes, got %d", expectedMemoryBytes, *res.Memory.Limit)
		}

		// PIDs Limit
		expectedPidLimit := int64(testRules.PidLimit)
		if res.Pids != nil && res.Pids.Limit != testRules.PidLimit {
			t.Errorf("Expected PIDs limit %d, got %d", expectedPidLimit, res.Pids.Limit)
		}

		// CPU Quota & Period
		if res.CPU != nil {
			// default cgroup period is usually 100000 microseconds
			expectedQuota := int64(testRules.CpuQuota)
			if *res.CPU.Quota != int64(expectedQuota) {
				t.Errorf("Expected CPU quota %d, got %d", expectedQuota, *res.CPU.Quota)
			}
		}
	} else {
		t.Error("Expected Linux Resources section to be defined for cgroup validation")
	}

	// ASSERTION 5: Storage Bind Mounts
	var foundCodeMount, foundTestsetMount bool
	for _, mount := range testOCISpecs.Mounts {
		if mount.Source == testRules.CodePathHost && mount.Destination == testRules.CodePathContainer {
			foundCodeMount = true
		}
		if mount.Source == testRules.TestsetPathHost && mount.Destination == testRules.TestsetPathContainer {
			foundTestsetMount = true
		}
	}

	if !foundCodeMount {
		t.Errorf("Missing code execution bind mount: %s -> %s", testRules.CodePathHost, testRules.CodePathContainer)
	}
	if !foundTestsetMount {
		t.Errorf("Missing testset context bind mount: %s -> %s", testRules.TestsetPathHost, testRules.TestsetPathContainer)
	}
}
