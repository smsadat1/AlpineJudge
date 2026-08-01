package internal

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	containerd "github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	oci "github.com/containerd/containerd/oci"
)

type WarmContainer struct {
	Container  containerd.Container
	Task       containerd.Task
	ContStatus <-chan containerd.ExitStatus
	SlotID     uint32
	Conn       net.Conn
	mu         sync.Mutex // Guard concurrent socket access
}

func CreateWarmContainer(ctx context.Context, client *containerd.Client, slotID uint32) (*WarmContainer, error) {

	// Pull the container image & build OCI specs
	alpineJudgeMasterImage := "ghcr.io/smsadat1/alpinejudge/master:test"
	image := getContainerImage(alpineJudgeMasterImage, client, ctx)

	// Guard against nil pointer image
	if image == nil {
		return nil, fmt.Errorf("containerd image object is nil for %s", image)
	}

	var opts []oci.SpecOpts
	opts = build_ociSpecOpts(slotID)

	// Initiate the container
	containerID := generateContainerID()
	snapshotID := containerID + "-snapshot"

	container, err := client.NewContainer(
		ctx,
		containerID,
		containerd.WithNewSnapshot(snapshotID, image),
		containerd.WithImage(image),
		containerd.WithNewSpec(opts...),

		// correct runtime name format requires full name of binary "runc" isn't enough
		containerd.WithRuntime("io.containerd.runc.v2", nil),
	)

	if err != nil {
		return nil, fmt.Errorf("Failed created container with ID %v", err)
	}

	// create related socket file on host before container boots
	socketPath := fmt.Sprintf("/tmp/runner/sockets/%d.sock", slotID)
	_ = os.Remove(socketPath) // clean stale socker

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create unix socket at %s: %w", socketPath, err)
	}
	defer listener.Close()

	_ = os.Chmod(socketPath, 0666) // permission for non-root

	// prepare container
	var stdoutWriter, stderrWrite bytes.Buffer

	// start container task
	task, err := container.NewTask(
		ctx,
		cio.NewCreator(cio.WithStreams(nil, &stdoutWriter, &stderrWrite)),
	)
	if err != nil {
		_ = container.Delete(ctx, containerd.WithSnapshotCleanup)
		return nil, fmt.Errorf("NewTask error: %w", err)
	}

	// obtain wait channel
	statusC, err := task.Wait(ctx)
	if err != nil {
		_, _ = task.Delete(ctx)
		_ = container.Delete(ctx, containerd.WithSnapshotCleanup)
		return nil, fmt.Errorf("failed to obtain wait channel: %w", err)
	}

	// Boot container process
	if err := task.Start(ctx); err != nil {
		_, _ = task.Delete(ctx)
		_ = container.Delete(ctx, containerd.WithSnapshotCleanup)
		return nil, fmt.Errorf("failed to start container task: %w", err)
	}

	conn, err := listener.Accept()
	if err != nil {
		_ = container.Delete(ctx, containerd.WithSnapshotCleanup)
		return nil, fmt.Errorf("agent failed to connect: %w", err)
	}

	// log.Printf("Successfully initiated container with ID %s and snapshot with ID %v", container.ID(), snapshotID)
	wc := WarmContainer{
		Container:  container,
		Task:       task,
		ContStatus: statusC,
		SlotID:     slotID,
		Conn:       conn,
	}
	return &wc, nil
}
