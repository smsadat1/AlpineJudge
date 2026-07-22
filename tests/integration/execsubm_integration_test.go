package integration

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

func Test_Execsubm_integration_test(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	testID := "test-123"

	req := testcontainers.ContainerRequest{
		Image:         "ghcr.io/smsadat1/alpinejudge/gcc:test",
		ImagePlatform: "linux/amd64",
		Env: map[string]string{
			"CONFIG_PATH":        "/workspace/execspec.json",
			"TESTSET_PATH":       "/workspace/" + testID + "/",
			"STREAM_SOCKET_PATH": "/workspace/agentstream.sock",
		},
		WorkingDir: "/workspace",
	}
	// testcontainers.Mounts()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
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
