package internal

import (
	"assert"
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
	"utils"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/v2/pkg/namespaces"
)

func Test_Build_ociSpecOpts(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	ctx = namespaces.WithNamespace(ctx, "test_build_oci_spec")
	defer cancel()

	var dummyContainer containers.Container
	var dummyClient oci.Client

	testSlotID := 420
	expectedArgs := []string{"/usr/bin/ajagent"}
	expectedEnv := []string{
		"PATH=/opt/java/openjdk/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"STREAM_SOCKET_PATH=/workspace/sockets/" + fmt.Sprint(testSlotID) + ".sock",
		"JAVA_HOME=/opt/java/openjdk",
		"GOROOT=/usr/local/go",
		"GOCACHE=/tmp/go-build",
	}

	// godotenv.Load(".env")

	t.Setenv("READONLY_ROOTFS", "true")
	t.Setenv("NO_NEW_PRIVILEGES", "true")
	t.Setenv("CPU_QUOTA", "2.0")
	t.Setenv("MEMORY_LIMIT_MB", "1024")
	t.Setenv("PID_LIMIT", "128")

	expectedRORFS, _ := strconv.ParseBool(os.Getenv("READONLY_ROOTFS"))
	expectedNNP, _ := strconv.ParseBool(os.Getenv("NO_NEW_PRIVILEGES"))
	expectedcpuQuota, _ := strconv.ParseFloat(os.Getenv("CPU_QUOTA"), 64)
	expectedMemLimitMB, _ := strconv.ParseUint(os.Getenv("MEMORY_LIMIT_MB"), 10, 64)
	expectedPidLimit, _ := strconv.ParseInt(os.Getenv("PID_LIMIT"), 10, 64)
	var tmpfsMountFound, workspaceMountFound bool

	testOpts := build_ociSpecOpts(uint32(testSlotID))

	testOCISpecs, err := oci.GenerateSpec(ctx, dummyClient, &dummyContainer, testOpts...)
	if err != nil {
		t.Fatal(err)
	}

	assert.Slice(t, testOCISpecs.Process.Args, expectedArgs)
	assert.Slice(t, testOCISpecs.Process.Env, expectedEnv)
	assert.Bool(t, expectedRORFS, testOCISpecs.Root.Readonly)
	assert.Bool(t, expectedNNP, testOCISpecs.Process.NoNewPrivileges)

	// Linux Kernel Resource Constraints (cgroups)
	if testOCISpecs.Linux != nil && testOCISpecs.Linux.Resources != nil {
		res := testOCISpecs.Linux.Resources

		// Memory Limit (OCI expects bytes: MB * 1024 * 1024)
		expectedMemoryBytes := int64(expectedMemLimitMB * 1024 * 1024)
		assert.Int64(t, expectedMemoryBytes, *res.Memory.Limit)
		assert.Int64(t, expectedMemoryBytes, *res.Memory.Swap)

		// PIDs Limit
		expectedPidLimit := int64(expectedPidLimit)
		assert.Int64(t, expectedPidLimit, res.Pids.Limit)

		// default cgroup period is usually 100000 microseconds
		expectedQuota := int64(expectedcpuQuota) * 100000
		// warn: values passed here are int64, risks of loosing precision in future
		assert.Int64(t, expectedQuota, *res.CPU.Quota)
	} else {
		t.Error("Expected Linux Resources section to be defined for cgroup validation")
	}

	// Validate tmpfs & workspace mounts
	for _, mount := range testOCISpecs.Mounts {
		if mount.Destination == "/tmp" {
			assert.String(t, "tmpfs", mount.Type)
			assert.String(t, "tmpfs", mount.Source)
			assert.Slice(t, mount.Options, []string{"nosuid", "nodev", "mode=1777"})
			tmpfsMountFound = true
		}
		if mount.Destination == "/workspace" {
			assert.String(t, "bind", mount.Type)
			assert.String(t, "/tmp/runner/", mount.Source)
			assert.Slice(t, mount.Options, []string{"rbind", "rw"})
			workspaceMountFound = true
		}
	}

	assert.Bool(t, true, workspaceMountFound)
	assert.Bool(t, true, tmpfsMountFound)
}

func Test_Build_AgentExecSpec(t *testing.T) {

	testRules := utils.ExecRules{
		SubmissionID:         "sub999",
		PerTestLogLimitKB:    512,
		PerTestTimeoutsec:    5,
		PerTestMemoryLimitMB: 1024,
		TestID:               "ts999",
		CompileArgs:          []string{"gcc", "-std=c11", "-Wall", "-Wextra", "-o", "/tmp/main", "/workspace/submissions/sub999/main.c"},
		RunArgs:              []string{"/tmp/main"},
	}

	testAgentSpec := Build_AgentExecSpec(testRules)

	assert.String(t, testAgentSpec.SubmissionID, testAgentSpec.SubmissionID)
	assert.Uint32(t, testRules.PerTestLogLimitKB, testAgentSpec.LogLimitKB)
	assert.Uint32(t, testRules.PerTestTimeoutsec, testAgentSpec.TimeoutSec)
	assert.Uint64(t, testRules.PerTestMemoryLimitMB, testAgentSpec.MemoryLimitMB)
	assert.String(t, "/workspace/testsets/"+testRules.TestID+"/", testAgentSpec.TestSetPath)
	assert.Slice(t, testRules.CompileArgs, testAgentSpec.CompileArgs)
	assert.Slice(t, testRules.RunArgs, testAgentSpec.RunArgs)
}
