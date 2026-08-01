package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"shared"
	"utils"
)

// manages entire container lifecycle
func OrchestrateSubm(
	ctx context.Context,
	cCtx context.Context,
	wc *WarmContainer,
	s3m shared.S3Manager,
	jobspec shared.JobSpec,
	rmqm shared.RMQManager,
) (utils.ContainerInfo, error) {

	// download testsets from S3
	if err := s3m.DownloadDirFromS3(ctx, jobspec.Bucket, jobspec.TestsetS3Key, "/tmp/"+jobspec.SubmissionID+"/"); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to download testet from S3:  %v\n", err)
	}

	// prepare execution rules
	err, rules := prepareExecrules(ctx, s3m, jobspec)
	if err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to generate execution rules: %v\n", err)
	}

	// prepare in-container agent specification and pass it via unix socket
	err, data := build_agentExecSpec(rules)
	if err != nil {
		return utils.ContainerInfo{}, err
	}

	dataBuffer := bytes.NewReader(data)
	if _, err := io.Copy(wc.Conn, dataBuffer); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("failed streaming exec spec body: %w", err)
	}

	// send delimiter byte so the agent knows the frame is finished
	if _, err := wc.Conn.Write([]byte{'\n'}); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("failed sending frame delimiter: %w", err)
	}

	// temp dir creation
	targetDir := filepath.Join("/tmp", jobspec.SubmissionID)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to create directory %s: %v\n", targetDir, err)
	}

	// 4. Manage the running continer, run tests & destroy before exit
	contInfo := wc.execSubm(cCtx, rules, jobspec, rmqm, s3m)

	return contInfo, nil
}
