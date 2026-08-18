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

		compileArgs = append(compileArgs, "-Wall")
		compileArgs = append(compileArgs, "-Wextra")
		compileArgs = append(compileArgs, "-o")
		compileArgs = append(compileArgs, "/tmp/main")
		compileArgs = append(compileArgs, submmittedCodePathBase+"main."+jobspec.Language)

		runArgs = append(runArgs, "/tmp/main")
	}

	if language == "go" {
		// compile once the run native Go binary for faster runtime

		compileArgs = append(compileArgs, "go", "build", "-o")
		compileArgs = append(compileArgs, submmittedCodePathBase+"main")    // Output binary path
		compileArgs = append(compileArgs, submmittedCodePathBase+"main.go") // Go file

		runArgs = append(runArgs, submmittedCodePathBase+"main")
	}

	if language == "java" {
		compileArgs = append(compileArgs, "javac", "-d")
		compileArgs = append(compileArgs, submmittedCodePathBase)
		compileArgs = append(compileArgs, submmittedCodePathBase+"Main.java")

		// compiled once and used fast JIT startup flag for faster runtime
		runArgs = append(runArgs, "java")
		runArgs = append(runArgs, "-XX:+TieredCompilation")
		runArgs = append(runArgs, "-XX:TieredStopAtLevel=1")
		runArgs = append(runArgs, "-cp", submmittedCodePathBase)
		runArgs = append(runArgs, "Main")
	}

	if language == "node" {
		runArgs = append(runArgs, "node")
		runArgs = append(runArgs, submmittedCodePathBase+"main.js")
	}

	if language == "py" {
		runArgs = append(runArgs, "python")
		runArgs = append(runArgs, submmittedCodePathBase+"main.py")
	}

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
