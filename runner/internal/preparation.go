package internal

import (
	"context"
	"utils"

	"shared"
)

func PrepareExecrules(
	ctx context.Context, s3m shared.S3Manager, jobspec shared.JobSpec,
) (error, utils.ExecRules) {

	language := jobspec.Language
	version := jobspec.Version

	var compileArgs []string
	var runArgs []string
	submmittedCodePathBase := "/workspace/submissions/" + jobspec.SubmissionID + "/"

	if language == "c" || language == "cpp" || language == "cc" {
		if language == "c" {
			compileArgs = append(compileArgs, "gcc")
		}

		if language == "cpp" || language == "cc" {
			compileArgs = append(compileArgs, "g++")
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
		compileArgs = append(compileArgs, "/tmp/main")
		compileArgs = append(compileArgs, submmittedCodePathBase+"main."+jobspec.Language)

		runArgs = append(runArgs, "/tmp/main")
	}

	if language == "go" {
		switch version {
		case "go1.24":
			runArgs = append(runArgs, "/usr/local/go1.24/bin/go")
		case "go1.26":
			runArgs = append(runArgs, "/usr/local/go1.26/bin/go")
		}
		runArgs = append(runArgs, "run")
		runArgs = append(runArgs, submmittedCodePathBase+"main.go")
	}

	if language == "java" {
		switch version {
		case "java25":
			compileArgs = append(compileArgs, "/usr/lib/jvm/java-25-openjdk/bin/javac")
		case "java26":
			compileArgs = append(compileArgs, "/usr/lib/jvm/java-26-openjdk/bin/javac")
		}
		compileArgs = append(compileArgs, submmittedCodePathBase+"Main.java")

		switch version {
		case "java25":
			runArgs = append(runArgs, "/usr/lib/jvm/java-25-openjdk/bin/java")
		case "java26":
			runArgs = append(runArgs, "/usr/lib/jvm/java-26-openjdk/bin/java")
		}
		runArgs = append(runArgs, "/workspace/Main")
	}

	if language == "node" {
		switch version {
		case "node18":
			runArgs = append(runArgs, "/usr/bin/node18")
		case "node22":
			runArgs = append(runArgs, "/usr/bin/node22")
		}
		runArgs = append(runArgs, submmittedCodePathBase+"main.js")
	}

	if language == "py" {
		switch version {
		case "python3.10":
			runArgs = append(runArgs, "/usr/bin/python3.10")
		case "python3.12":
			runArgs = append(runArgs, "/usr/bin/python3.12")
		}
		runArgs = append(runArgs, submmittedCodePathBase+"main.py")
	}

	containerImage := jobspec.Image
	hostSrcFilePath := "/tmp/runner/submissions/" + jobspec.SubmissionID + "/main." + language
	hostTestFileDir := "/tmp/runner/testsets/" + jobspec.Testset + "/"

	err := downloadFileS3(
		ctx, s3m,
		jobspec.Bucket,
		jobspec.SrcCodeS3Key,
		jobspec.TestsetS3Key,
		hostSrcFilePath,
		hostTestFileDir,
	)

	if err != nil {
		return err, utils.ExecRules{}
	}

	execRules := utils.ExecRules{
		// unique container ID to avoid collision
		ContainerID: generateContainerID(),

		Image:       containerImage,
		CompileArgs: compileArgs,
		RunArgs:     runArgs,

		TestID:         jobspec.Testset,
		EventQueueName: jobspec.SSEQueue,

		PerTestMemoryLimitMB: jobspec.PerTestMemoryLimitMB,
		PerTestTimeoutsec:    jobspec.PerTestTimeoutsec,
		PerTestLogLimitKB:    jobspec.PerTestLogLimitKB,
	}

	return nil, execRules
}
