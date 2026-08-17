package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"local/runner/internal"
	"log"
	"os"
	"path/filepath"
	"shared"
	"utils"
)

func isTestsetDirEmpty(testsetPath string) (bool, error) {

	testSetDir, err := os.Open(testsetPath)
	if err != nil {
		return false, err
	}
	defer testSetDir.Close()

	if _, err := testSetDir.Readdirnames(1); err != nil {
		return false, err
	}
	return true, nil
}

// manages entire container lifecycle
func OrchestrateSubm(
	ctx context.Context,
	cCtx context.Context,
	wc *internal.WarmContainer,
	s3m shared.S3Manager,
	jobspec shared.JobSpec,
	rmqm shared.RMQManager,
) (utils.ContainerInfo, error) {

	// submission specific sub dircectories creation
	dirs := []string{
		filepath.Join("/tmp", "runner", "testsets", jobspec.Testset),
		filepath.Join("/tmp", "runner", "submissions", jobspec.SubmissionID),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return utils.ContainerInfo{}, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// check if testset already exists on filesystem
	testsetDirPath := fmt.Sprintf("/tmp/runner/testsets/%v/", jobspec.Testset)
	exists, _ := isTestsetDirEmpty(testsetDirPath)

	// download testsets from S3 only when it's not already downloaded
	if !exists {
		log.Printf("Testset %v doesn't exists on filesystem, downloading...\n", testsetDirPath)
		if err := s3m.DownloadDirFromS3(ctx,
			jobspec.Bucket, jobspec.TestsetS3Key, testsetDirPath); err != nil {
			return utils.ContainerInfo{}, fmt.Errorf("Failed to download testet from S3:  %v\n", err)
		}
	} else {
		log.Printf("Testset %v already exists on filesystem, skipping download...\n", testsetDirPath)
	}

	// prepare execution rules
	err, rules := internal.PrepareExecrules(ctx, s3m, jobspec)
	if err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to generate execution rules: %v\n", err)
	}

	// prepare in-container agent specification and pass it via unix socket
	ajagentSpec := internal.Build_AgentExecSpec(rules)

	if err := json.NewEncoder(wc.Conn).Encode(ajagentSpec); err != nil {
		if task, tErr := wc.Container.Task(ctx, nil); tErr == nil {
			st, _ := task.Status(cCtx)
			log.Printf("--> TASK STATE AT BROKEN PIPE: status=%v exitCode=%d pid=%d", st.Status, st.ExitStatus, task.Pid())
		} else {
			log.Printf("--> TASK COULD NOT BE FOUND (already deleted by containerd?): %v", tErr)
		}
		return utils.ContainerInfo{}, fmt.Errorf("Failed sending execspec JSON: %w", err)
	}

	// temp dir creation
	targetDir := filepath.Join("/tmprunner/submissions", jobspec.SubmissionID)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return utils.ContainerInfo{}, fmt.Errorf("Failed to create directory %s: %v\n", targetDir, err)
	}

	// 4. Manage the running continer, run tests & destroy before exit
	contInfo := wc.ExecSubm(cCtx, rules, jobspec, rmqm, s3m)

	return contInfo, nil
}
