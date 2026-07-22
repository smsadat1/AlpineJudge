package executor

import (
	"context"
	"fmt"
	"log"
	"os"
	"shared"
	"utils"

	containerd "github.com/containerd/containerd"
	oci "github.com/containerd/containerd/oci"
)

// manages entire container lifecycle
func ExecSubmission(
	ctx context.Context, client *containerd.Client, s3m shared.S3Manager, jobspec shared.JobSpec, rmqm shared.RMQManager,
) (utils.ResultSpec, error) {

	// 1. Prepare execution rules
	err, rules := PrepareExecrules(ctx, s3m, jobspec, false)
	if err != nil {
		return utils.ResultSpec{}, fmt.Errorf("Failed to generate execution rules\n")
	}

	// 2. Pull the container image & build OCI specs
	image := getContainerImage(rules.Image, client, ctx)
	var opts []oci.SpecOpts
	opts = Build_ociSpecOpts(rules)
	err, data := Build_agentExecSpec(rules)
	if err != nil {
		return utils.ResultSpec{}, err
	}

	if err := os.WriteFile("/tmp/execspec.json", data, os.ModeAppend); err != nil {
		return utils.ResultSpec{}, fmt.Errorf("Failed to create agent execspec json:  %v\n", err)
	}

	// 3. Initiate the container
	snapshotID := rules.ContainerID + "-snapshot"
	container, err := client.NewContainer(
		ctx,
		rules.ContainerID,
		containerd.WithNewSnapshot(snapshotID, image),
		containerd.WithImage(image),
		containerd.WithNewSpec(opts...),
		containerd.WithRuntime("runc", nil),
	)

	if err != nil {
		return utils.ResultSpec{}, fmt.Errorf("Failed created container with ID %s", container.ID())
	}
	log.Printf("Successfully initiated container with ID %s and snapshot with ID %v", container.ID(), snapshotID)

	// 4. Manage the running continer, run tests & destroy before exit
	result := ExecSubm(ctx, container, rules, jobspec, rmqm, s3m)

	return result, nil
}
