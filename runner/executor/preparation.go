package executor

import (
	"context"
	"utils"

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

func PrepareExecrules(
	ctx context.Context, s3m shared.S3Manager, jobspec shared.JobSpec,
	testMode bool, // used only for tests | must stay false for production
) (error, utils.ExecRules) {

	// submID := jobspec.SubmissionID
	language := jobspec.Language
	version := jobspec.Version
	testID := jobspec.Testset

	var compileArgs []string
	var runArgs []string

	if language == "c" || language == "cpp" || language == "cc" {
		if language == "c" {
			compileArgs = append(compileArgs, "/usr/bin/gcc")
		}

		if language == "cpp" || language == "cc" {
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
		switch version {
		case "node18":
			runArgs = append(runArgs, "/usr/bin/node18")
		case "node22":
			runArgs = append(runArgs, "/usr/bin/node22")
		}
		runArgs = append(runArgs, "main.js")
	}

	if language == "py" {
		switch version {
		case "python3.10":
			runArgs = append(runArgs, "/usr/bin/python3.10")
		case "python3.12":
			runArgs = append(runArgs, "/usr/bin/python3.12")
		}
		runArgs = append(runArgs, "main.py")
	}

	containerImage := utils.RunCfg.Images[language]
	hostWorkDir := "/tmp/ajrunner/" + utils.RunCfg.RunnerID + "/"
	hostSrcFilePath := hostWorkDir + "main." + language
	hostTestFileDir := hostWorkDir + testID

	containerSrcFilePath := "/workspace/main." + language
	containerTestFileDir := "/workspace/" + testID + "/" + jobspec.TestsetVersion + "/"

	// for other service's usage and testcase overrides
	HostSrcFilePath = hostSrcFilePath
	HostTestFilePath = hostTestFileDir

	if testMode {
		HostSrcFilePath = "../artifacts/main.cc"
		HostTestFilePath = "../artifacts/ts001"
	}

	err := downloadFileS3(
		ctx, s3m,
		jobspec.Bucket,
		jobspec.SrcCodeS3Key,
		jobspec.TestsetS3Key,
		HostSrcFilePath,
		HostTestFilePath,
	)

	if err != nil {
		return err, utils.ExecRules{}
	}

	execRules := utils.ExecRules{
		Image:       containerImage,
		CompileArgs: compileArgs,
		RunArgs:     runArgs,

		CodePathHost:         HostSrcFilePath,
		CodePathContainer:    containerSrcFilePath,
		TestsetPathHost:      HostTestFilePath,
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
