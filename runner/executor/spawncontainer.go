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
func OrchestrateSubm(
	ctx context.Context, client *containerd.Client, s3m shared.S3Manager, jobspec shared.JobSpec, rmqm shared.RMQManager, testMode bool,
) (utils.ContainerInfo, error) {

	// 1. Prepare execution rules
	err, rules := PrepareExecrules(ctx, s3m, jobspec, testMode)
	if err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to generate execution rules: %v\n", err)
	}

	// 2. Pull the container image & build OCI specs
	image := getContainerImage(rules.Image, client, ctx)

	// prepare in-container agent specification file
	err, data := Build_agentExecSpec(rules)
	if err != nil {
		return utils.ContainerInfo{}, err
	}
	if err := os.WriteFile("/tmp/"+jobspec.SubmissionID+"/execspec.json", data, os.ModeAppend); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to create agent execspec json:  %v\n", err)
	}

	// download testsets from S3
	if err := os.Mkdir("/tmp/"+jobspec.SubmissionID+"/"+jobspec.Testset+"/", os.FileMode(os.O_RDWR)); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to create temporary testset location:  %v\n", err)
	}
	if err := s3m.DownloadDirFromS3(ctx, jobspec.Bucket, jobspec.TestsetS3Key, "/tmp/"+jobspec.SubmissionID+"/"); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to download testet from S3:  %v\n", err)
	}

	var opts []oci.SpecOpts
	opts = Build_ociSpecOpts(rules)

	// 3. Initiate the container
	snapshotID := rules.ContainerID + "-snapshot"
	container, err := client.NewContainer(
		ctx,
		rules.ContainerID,
		containerd.WithNewSnapshot(snapshotID, image),
		containerd.WithImage(image),
		containerd.WithNewSpec(opts...),

		// correct runtime name format requires full name of binary "runc" isn't enough
		containerd.WithRuntime("io.containerd.runc.v2", nil),
	)

	if err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed created container with ID %v", err)
	}
	log.Printf("Successfully initiated container with ID %s and snapshot with ID %v", container.ID(), snapshotID)

	// 4. Manage the running continer, run tests & destroy before exit
	contInfo := ExecSubm(ctx, container, rules, jobspec, rmqm, s3m)

	return contInfo, nil
}
