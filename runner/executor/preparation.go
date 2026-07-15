package executor

import (
	"context"

	"local/runner/utils"
	"shared"
)

var (
	HostTestFilePath string
	HostSrcFilePath  string
)

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
	ctx context.Context, s3m shared.S3Manager, jobspec shared.JobSpec,
) (error, utils.ExecRules) {

	submID := jobspec.SubmissionID
	language := jobspec.Language
	version := jobspec.Version
	testID := jobspec.Testset + jobspec.TestsetVersion

	var compileArgs []string
	var runArgs []string
	var containerImage string

	if language == "c" || language == "cpp" {
		containerImage = "alpinejudge/gcc"

		if language == "c" {
			compileArgs = append(compileArgs, "/usr/bin/gcc")
		}

		if language == "cpp" {
			compileArgs = append(compileArgs, "/usr/bin/g++")
		}

		switch version {
		case "c99":
			compileArgs = append(compileArgs, "-std=c99")
		case "c11":
			compileArgs = append(compileArgs, "-std=c17")
		case "c17":
			compileArgs = append(compileArgs, "-std=c17")
		case "c++11":
			compileArgs = append(compileArgs, "-std=c++11")
		case "c++17":
			compileArgs = append(compileArgs, "-std=c++17")
		case "c++20":
			compileArgs = append(compileArgs, "-std=c++20")
		}

		compileArgs = append(compileArgs, "-Wall")
		compileArgs = append(compileArgs, "-Wextra")
		compileArgs = append(compileArgs, "-o")
		compileArgs = append(compileArgs, "main")
		compileArgs = append(compileArgs, "main."+jobspec.Language)

		runArgs = append(runArgs, "./main")
	}

	if language == "go" {
		containerImage = "alpinejudge/go"

		switch version {
		case "go1.24":
			runArgs = append(runArgs, "/usr/local/go1.24/bin/go")
		case "go1.26":
			runArgs = append(runArgs, "/usr/local/go1.26/bin/go")
		}
		runArgs = append(runArgs, "run")
		runArgs = append(runArgs, "main.go")
	}

	if language == "java" {
		containerImage = "alpinejudge/java"

		switch version {
		case "java25":
			compileArgs = append(compileArgs, "/usr/lib/jvm/java-25-openjdk/bin/javac")
		case "java26":
			compileArgs = append(compileArgs, "/usr/lib/jvm/java-26-openjdk/bin/javac")
		}
		compileArgs = append(compileArgs, "Main.java")

		switch version {
		case "java25":
			runArgs = append(runArgs, "/usr/lib/jvm/java-25-openjdk/bin/java")
		case "java26":
			runArgs = append(runArgs, "/usr/lib/jvm/java-26-openjdk/bin/java")
		}
		runArgs = append(runArgs, "Main")
	}

	if language == "node" {
		containerImage = "alpinejudge/node"

		switch version {
		case "node18":
			runArgs = append(runArgs, "/usr/bin/node18")
		case "node22":
			runArgs = append(runArgs, "/usr/bin/node22")
		}
		runArgs = append(runArgs, "main.js")
	}

	if language == "py" {
		containerImage = "alpinejudge/python"

		switch version {
		case "python3.10":
			runArgs = append(runArgs, "/usr/bin/python3.10")
		case "python3.12":
			runArgs = append(runArgs, "/usr/bin/python3.12")
		}
		runArgs = append(runArgs, "main.py")
	}

	hostWorkDir := "/tmp/ajrunner/" + "/"
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
		Image:       containerImage,
		CompileArgs: compileArgs,
		RunArgs:     runArgs,

		CodePathHost:         hostSrcFilePath,
		CodePathContainer:    containerSrcFilePath,
		TestsetPathHost:      hostTestFileDir,
		TestsetPathContainer: containerTestFileDir,

		CpuQuota:       float64(utils.RunCfg.Limits.CPUQuota),
		MemoryLimitMB:  utils.RunCfg.Limits.MemoryLimitMB,
		NoNewPrivilege: utils.RunCfg.Limits.NoNewPrivs,
		PidLimit:       int64(utils.RunCfg.Limits.PIDLimit),
		Timeoutsec:     uint32(utils.RunCfg.Limits.TimeoutSec),
		ReadOnlyRootfs: utils.RunCfg.Limits.RORootFS,
	}

	return nil, execRules
}
