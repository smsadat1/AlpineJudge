package executor

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"utils"

	nanoid "github.com/jaevor/go-nanoid"

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

func generateContainerID() string {
	nanoID, err := nanoid.CustomASCII("0123456789", 12)
	contID := nanoID()
	if err != nil {
		// fallback to random numbers
		contID = fmt.Sprint(rand.Intn(3000000))
	}
	return contID
}

func PrepareExecrules(
	ctx context.Context, s3m shared.S3Manager, jobspec shared.JobSpec,
	testMode bool, // used only for tests | must stay false for production
) (error, utils.ExecRules) {
	language := jobspec.Language
	version := jobspec.Version
	testID := jobspec.Testset

	var compileArgs []string
	var runArgs []string

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
		compileArgs = append(compileArgs, "/workspace/main."+jobspec.Language)

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
		runArgs = append(runArgs, "/workspace/main.go")
	}

	if language == "java" {
		switch version {
		case "java25":
			compileArgs = append(compileArgs, "/usr/lib/jvm/java-25-openjdk/bin/javac")
		case "java26":
			compileArgs = append(compileArgs, "/usr/lib/jvm/java-26-openjdk/bin/javac")
		}
		compileArgs = append(compileArgs, "/workspace/Main.java")

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
		runArgs = append(runArgs, "/workspace/main.js")
	}

	if language == "py" {
		switch version {
		case "python3.10":
			runArgs = append(runArgs, "/usr/bin/python3.10")
		case "python3.12":
			runArgs = append(runArgs, "/usr/bin/python3.12")
		}
		runArgs = append(runArgs, "/workspace/main.py")
	}

	containerImage := jobspec.Image
	hostWorkDir := "/tmp/" + jobspec.SubmissionID + "/"
	hostSrcFilePath := hostWorkDir + "main." + language
	hostTestFileDir := hostWorkDir + testID

	// containerSrcFilePath := "/workspace/main." + language
	containerTestFileDir := "/workspace/"

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

	// create & prepare dedicated tmp dir
	tempLocation := "/tmp/" + jobspec.SubmissionID + "/"
	if err := os.MkdirAll(tempLocation, os.FileMode(os.O_RDWR)); err != nil {
		return fmt.Errorf("Failed to create temporary directory: %v", err), utils.ExecRules{}
	}

	// create source file on host
	tempSrc := tempLocation + "main." + language
	if err := os.WriteFile(tempSrc, []byte(jobspec.Source), 0644); err != nil {
		return fmt.Errorf("write source: %w", err), utils.ExecRules{}
	}

	// create event stream socket (agent.sock)
	tempSock := tempLocation + "agent.sock"
	log.Printf("%v", compileArgs)

	execRules := utils.ExecRules{
		// unique container ID to avoid collision
		ContainerID: generateContainerID(),

		Image:                      containerImage,
		CompileArgs:                compileArgs,
		RunArgs:                    runArgs,
		CodePathHost:               tempLocation + "main." + language,
		CodePathContainer:          "/workspace/main." + language,
		TestsetPathHost:            HostTestFilePath,
		TestsetPathContainer:       containerTestFileDir,
		ExecutionSpecPathHost:      "/tmp/" + jobspec.SubmissionID + "/execspec.json",
		ExecutionSpecPathContainer: "/workspace/execspec.json",
		HostEventSocket:            tempSock,
		ContainerEventSocket:       "/tmp/agent.sock", // keep socket separate from workspace to keep socket more reliable
		EventQueueName:             jobspec.EventQueue,

		CpuQuota:       float64(utils.RunCfg.Limits.CPUQuota),
		MemoryLimitMB:  utils.RunCfg.Limits.MemoryLimitMB,
		NoNewPrivilege: utils.RunCfg.Limits.NoNewPrivs,
		PidLimit:       int64(utils.RunCfg.Limits.PIDLimit),
		Timeoutsec:     uint32(utils.RunCfg.Limits.TimeoutSec),
		ReadOnlyRootfs: utils.RunCfg.Limits.RORootFS,
	}

	return nil, execRules
}
