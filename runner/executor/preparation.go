package executor

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"local/runner/utils"
	"shared"
)

var (
	HostTestFilePath string
	HostSrcFilePath  string
)

func loadResLimits(configdata []byte, execr *utils.ExecRules) error {

	if err := yaml.Unmarshal(configdata, &execr); err != nil {
		return fmt.Errorf("Failed to unmarshal config\n")
	}
	return nil
}

func downloadFileS3(
	ctx context.Context, s3m shared.S3Manager,
	bucket string, srcCodeS3key string, testsetS3key string,
	hostSrcFilePath string, hostTestFileDir string,
) error {

	if err := s3m.DownloadFileFromS3(ctx, bucket, srcCodeS3key, hostSrcFilePath); err != nil {
		return err
	}

	if err := s3m.DownloadDirFromS3(ctx, bucket, testsetS3key, hostTestFileDir); err != nil {
		return err
	}

	return nil
}

func prepareExecrules(
	ctx context.Context, s3m shared.S3Manager, configPath string, jobspec utils.JobSpec,
) (error, utils.ExecRules) {

	jobID := jobspec.JobId
	submID := jobspec.SubmissionID
	language := jobspec.Language
	version := jobspec.Version
	testID := jobspec.Testset + jobspec.TestsetVersion

	var cmdArgs []string
	var containerImage string

	if language == "c" || language == "cpp" {
		containerImage = "alpinejudge/gcc"

		if language == "c" {
			cmdArgs = append(cmdArgs, "/usr/bin/gcc")
		}

		if language == "cpp" {
			cmdArgs = append(cmdArgs, "/usr/bin/g++")
		}

		switch version {
		case "c99":
			cmdArgs = append(cmdArgs, "-std=c99")
		case "c11":
			cmdArgs = append(cmdArgs, "-std=c17")
		case "c17":
			cmdArgs = append(cmdArgs, "-std=c17")
		case "c++11":
			cmdArgs = append(cmdArgs, "-std=c++11")
		case "c++17":
			cmdArgs = append(cmdArgs, "-std=c++17")
		case "c++20":
			cmdArgs = append(cmdArgs, "-std=c++20")
		}

		cmdArgs = append(cmdArgs, "-Wall")
		cmdArgs = append(cmdArgs, "-Wextra")
		cmdArgs = append(cmdArgs, "-o")
		cmdArgs = append(cmdArgs, "main")
		cmdArgs = append(cmdArgs, "main."+jobspec.Language)
	}

	if language == "go" {
		containerImage = "alpinejudge/go"

		switch version {
		case "go1.24":
			cmdArgs = append(cmdArgs, "/usr/local/go1.24/bin/go")
		case "go1.26":
			cmdArgs = append(cmdArgs, "/usr/local/go1.26/bin/go")
		}
		cmdArgs = append(cmdArgs, "run")
		cmdArgs = append(cmdArgs, "main.go")
	}

	if language == "java" {
		containerImage = "alpinejudge/java"

		switch version {
		case "java25":
			cmdArgs = append(cmdArgs, "/usr/lib/jvm/java-25-openjdk/bin/javac")
		case "java26":
			cmdArgs = append(cmdArgs, "/usr/lib/jvm/java-26-openjdk/bin/javac")
		}
		cmdArgs = append(cmdArgs, "Main.java")

		switch version {
		case "java25":
			cmdArgs = append(cmdArgs, "/usr/lib/jvm/java-25-openjdk/bin/java")
		case "java26":
			cmdArgs = append(cmdArgs, "/usr/lib/jvm/java-26-openjdk/bin/java")
		}
		cmdArgs = append(cmdArgs, "Main")
	}

	if language == "node" {
		containerImage = "alpinejudge/node"

		switch version {
		case "node18":
			cmdArgs = append(cmdArgs, "/usr/bin/node18")
		case "node22":
			cmdArgs = append(cmdArgs, "/usr/bin/node22")
		}
		cmdArgs = append(cmdArgs, "main.js")
	}

	if language == "py" {
		containerImage = "alpinejudge/python"

		switch version {
		case "python3.10":
			cmdArgs = append(cmdArgs, "/usr/bin/python3.10")
		case "python3.12":
			cmdArgs = append(cmdArgs, "/usr/bin/python3.12")
		}
		cmdArgs = append(cmdArgs, "main.py")
	}

	hostWorkDir := "/tmp/ajrunner/" + jobID + "/"
	hostSrcFilePath := hostWorkDir + submID + "." + language
	hostTestFileDir := hostWorkDir + testID

	containerSrcFilePath := "/workspace/main." + language
	containerTestFileDir := "/workspace/" + testID

	err := downloadFileS3(
		ctx, s3m, jobspec.Bucket, jobspec.SrcCodeS3Key, jobspec.TestsetS3Key, hostSrcFilePath, hostTestFileDir,
	)

	if err != nil {
		return err, utils.ExecRules{}
	}

	// for other service's usage
	HostSrcFilePath = hostSrcFilePath
	HostTestFilePath = hostTestFileDir

	execRules := utils.ExecRules{
		ContainerID:          jobID,
		Args:                 cmdArgs,
		Image:                containerImage,
		CodePathHost:         hostSrcFilePath,
		CodePathContainer:    containerSrcFilePath,
		TestsetPathHost:      hostTestFileDir,
		TestsetPathContainer: containerTestFileDir,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("Failed to open config(%v): %v", configPath, err), execRules
	}

	loadResLimits(data, &execRules)

	return nil, execRules
}
