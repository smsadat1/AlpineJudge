package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"local/runner/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var WA = fmt.Errorf("Wrong answer")
var IE = fmt.Errorf("Internal error")
var OLE = errors.New("Log limit exceeded")
var TLE = fmt.Errorf("Time limit exceeded")

type LimitExceededWriter struct {
	buf   bytes.Buffer
	limit int64
}

func (w *LimitExceededWriter) Write(p []byte) (int, error) {
	if int64(w.buf.Len()+len(p)) > w.limit {
		return 0, OLE
	}
	return w.buf.Write(p)
}

func runTestCase(
	spec utils.AgentExecSpec, inputPath string, expectedPath string,
) error {

	// per testcase context & timeout
	tcCtx, tcCancel := context.WithTimeout(
		context.Background(),
		time.Duration(spec.TimeoutSec)*time.Second,
	)
	defer tcCancel()

	stdin, err := os.Open(inputPath)
	if err != nil {
		return IE
	}
	defer stdin.Close()

	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		return IE
	}

	if len(spec.RunArgs) == 0 {
		return IE
	}

	cmd := exec.CommandContext(tcCtx, spec.RunArgs[0], spec.RunArgs[1:]...)
	cmd.Stdin = stdin

	stdout := &LimitExceededWriter{limit: int64(spec.LogLimitKB) * 1000}
	stderr := &LimitExceededWriter{limit: int64(spec.LogLimitKB) * 1000}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Run()
	if err != nil {
		if errors.Is(err, OLE) {
			return OLE
		}

		if tcCtx.Err() == context.DeadlineExceeded {
			return TLE
		}

		return IE
	}

	actualOut := strings.TrimSpace(stdout.buf.String())
	wantedOut := strings.TrimSpace(string(expected))

	if actualOut != wantedOut {
		return WA
	}

	return nil
}

// in-container agent to run execution
func RunnerAgent() {

	// find & load /workspace/execspec.json to spec
	jsonData, err := os.ReadFile("/workspace/execspec.json")
	if err != nil {
		log.Fatalf("Failed to load execspec %v\n", err)
	}

	var execSpec utils.AgentExecSpec
	if err := json.Unmarshal(jsonData, &execSpec); err != nil {
		log.Fatalf("Failed to unmarshal execspec %v\n", err)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(execSpec.TimeoutSec)*time.Second,
	)
	defer cancel()

	// compile stage
	if execSpec.HasCompile {
		cmd := exec.CommandContext(
			ctx, execSpec.CompileArgs[0],
			execSpec.CompileArgs[1:]...,
		)

		stdout := &LimitExceededWriter{limit: int64(execSpec.LogLimitKB) * 1000}
		stderr := &LimitExceededWriter{limit: int64(execSpec.LogLimitKB) * 1000}

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		err := cmd.Run()
		if err != nil {
			if errors.Is(err, OLE) {
				log.Fatal(OLE)
			}
			log.Fatal(err)
		}
	}

	// run stage
	entries, err := os.ReadDir(execSpec.TestSetPath)
	if err != nil {
		log.Fatal(err)
	}

	for _, ts := range entries {

		if !ts.IsDir() {
			continue
		}

		testcaseDir := filepath.Join(execSpec.TestSetPath, ts.Name())
		inputPath := filepath.Join(testcaseDir + "in.txt")
		expectedPath := filepath.Join(testcaseDir + "out.txt")

		if err := runTestCase(execSpec, inputPath, expectedPath); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	RunnerAgent()
}
