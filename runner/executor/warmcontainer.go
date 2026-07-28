package executor

import (
	"context"
	"fmt"
	"log"

	containerd "github.com/containerd/containerd"
	oci "github.com/containerd/containerd/oci"
	"github.com/jaevor/go-nanoid"
)

func CreateWarmContainer(ctx context.Context, client *containerd.Client) (containerd.Container, error) {

	// Pull the container image & build OCI specs
	alpineJudgeMasterImage := "ghcr.io/smsada1/alpinejudge/master:test"
	image := getContainerImage(alpineJudgeMasterImage, client, ctx)

	var opts []oci.SpecOpts
	opts = Build_ociSpecOpts()
	genContainerdID, err := nanoid.Standard(8)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate containre ID %v", err)
	}

	containerID := genContainerdID()

	// Initiate the container
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

	log.Printf("Successfully initiated container with ID %s and snapshot with ID %v", container.ID(), snapshotID)
	return container, nil
}
