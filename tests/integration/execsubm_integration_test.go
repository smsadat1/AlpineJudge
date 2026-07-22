package integration

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

func Test_Execsubm_integration_test(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// testID := "test-123"
	absConfigPath, err := filepath.Abs("../artifacts/execspec1.json")
	if err != nil {
		t.Fatalf("failed to resolve absolute path for execspec1.json: %v", err)
	}
	absFilePath, err := filepath.Abs("../artifacts/main.cpp")
	if err != nil {
		t.Fatalf("failed to resolve absolute path for main.cpp: %v", err)
	}

	// Setup unix socket
	// _ = os.Remove("../artifacts/agent.sock") // cleanup stale socket
	// _, err = net.Listen("unix", "../artifacts/agent.sock")
	// if err != nil {
	// 	log.Fatalf("Failed to create socket listener: %v", err)
	// }

	// absSockPath, err := filepath.Abs("../artifacts/agent.sock")
	// if err != nil {
	// 	t.Fatalf("failed to resolve absolute path for agent.sock: %v", err)
	// }

	req := testcontainers.ContainerRequest{
		Image:         "ghcr.io/smsadat1/alpinejudge/gcc:test",
		ImagePlatform: "linux/amd64",
		Env: map[string]string{
			"CONFIG_PATH":        "/workspace/execspec.json",
			"TESTSET_PATH":       "/workspace/test123/",
			"STREAM_SOCKET_PATH": "/workspace/agent.sock",
		},
		WorkingDir: "/workspace",

		Files: []testcontainers.ContainerFile{
			// {
			// 	HostFilePath:      "../artifacts/ts001",  // on host
			// 	ContainerFilePath: "/workspace/test123/", // in container
			// 	FileMode:          0644,
			// },
			{
				HostFilePath:      absConfigPath,              // on host
				ContainerFilePath: "/workspace/execspec.json", // inside container
				FileMode:          0644,
			},
			{
				HostFilePath:      absFilePath,            // on host
				ContainerFilePath: "/workspace/main1.cpp", // inside container
				FileMode:          0644,
			},
			// {
			// 	HostFilePath:      absSockPath,             // on host
			// 	ContainerFilePath: "/workspace/agent.sock", // inside container
			// 	FileMode:          0644,
			// },
		},
	}
	// testcontainers.Mounts()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	// If it failed to start or crashed immediately, print the container logs!
	if container != nil {
		logs, logErr := container.Logs(ctx)
		if logErr == nil {
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(logs)
			t.Logf("=== CONTAINER CRASH LOGS ===\n%s\n===========================", buf.String())
		}
	}
	if err != nil {
		t.Fatalf("failed to start agent container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	// 2. Fetch container logs to verify ajagent booted up and opened the socket
	logs, err := container.Logs(ctx)
	if err == nil {
		logBuf := new(bytes.Buffer)
		_, _ = logBuf.ReadFrom(logs)
		t.Logf("Agent startup logs:\n%s", logBuf.String())
	}

	// 3. Now exec commands against the running agent environment
	exitCode, reader, err := container.Exec(ctx, []string{"gcc", "--version"})
	if err != nil {
		t.Fatalf("failed to exec gcc: %v", err)
	}

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(reader)
	t.Logf("GCC Exit Code: %d", exitCode)
	t.Logf("GCC Output:\n%s", buf.String())
}
