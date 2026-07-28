package executor

import (
	"context"
	"fmt"
	"os"
	"shared"
	"utils"

	containerd "github.com/containerd/containerd"
)

// manages entire container lifecycle
func OrchestrateSubm(
	ctx context.Context, container containerd.Container, s3m shared.S3Manager, jobspec shared.JobSpec, rmqm shared.RMQManager, testMode bool,
) (utils.ContainerInfo, error) {

	// 1. Prepare execution rules
	err, rules := PrepareExecrules(ctx, s3m, jobspec, testMode)
	if err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to generate execution rules: %v\n", err)
	}

	// prepare in-container agent specification file
	err, data := Build_agentExecSpec(rules)
	if err != nil {
		return utils.ContainerInfo{}, err
	}
	if err := os.WriteFile("/tmp/"+jobspec.SubmissionID+"/execspec.json", data, os.ModeAppend); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to create agent execspec json:  %v\n", err)
	}

	// download testsets from S3
	// if err := os.Mkdir("/tmp/"+jobspec.SubmissionID+"/"+jobspec.Testset+"/", os.FileMode(os.O_RDWR)); err != nil {
	// 	return utils.ContainerInfo{}, fmt.Errorf("Failed to create temporary testset location:  %v\n", err)
	// }
	if err := s3m.DownloadDirFromS3(ctx, jobspec.Bucket, jobspec.TestsetS3Key, "/tmp/"+jobspec.SubmissionID+"/"); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to download testet from S3:  %v\n", err)
	}

	// 4. Manage the running continer, run tests & destroy before exit
	contInfo := ExecSubm(ctx, container, rules, jobspec, rmqm, s3m)

	return contInfo, nil
}
