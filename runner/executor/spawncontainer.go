package executor

import (
	"context"
	"fmt"
	"log"
	"shared"

	containerd "github.com/containerd/containerd/v2/client"

	"local/runner/utils"
)

// manages entire container lifecycle
func ExecSubmission(
	ctx context.Context, client *containerd.Client, s3m shared.S3Manager, configPath string, jobspec utils.JobSpec,
) (utils.ResultSpec, error) {

	// 1. Prepare execution rules
	err, rules := prepareExecrules(ctx, s3m, configPath, jobspec)
	if err != nil {
		return utils.ResultSpec{}, fmt.Errorf("Failed to generate execution rules\n")
	}

	// 2. Pull the container image & build OCI specs
	image := getContainerImage(rules.Image, client, ctx)
	opts := build_ociSpecOpts(image, rules)

	// 3. Initiate the container
	snapshotID := rules.ContainerID + "-snapshot"
	container, err := client.NewContainer(
		ctx,
		rules.ContainerID,
		containerd.WithNewSnapshot(snapshotID, image),
		containerd.WithNewSpec(opts...),
		containerd.WithRuntime("runsc", nil), // gVisor interception
	)

	if err != nil {
		return utils.ResultSpec{}, fmt.Errorf("Failed created container with ID %s", container.ID())
	}
	log.Printf("Successfully initiated container with ID %s and snapshot with ID %v", container.ID(), snapshotID)

	// 4. Manage the running continer & destroy before exit
	result, err := manageContainer(ctx, container, rules)
	if err != nil {
		return utils.ResultSpec{}, err
	}

	return result, nil
}
