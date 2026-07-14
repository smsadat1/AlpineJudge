package images

import (
	"encoding/json"
	"fmt"
	"local/runner/utils"
	"log"
	"os"
	"os/exec"
)

// in-container agent to run execution

func RunnerAgent() {

	// find & load /workspace/execspec.json to spec
	jsonData, err := os.ReadFile("/workspace/execspec.json")
	if err != nil {
		log.Fatal(err)
	}

	var execSpec utils.AgentExecSpec

	if err := json.Unmarshal(jsonData, &execSpec); err != nil {
		log.Fatal(err)
	}

	// execute command & stream output to stdout
	if execSpec.HasCompilePhase {
		cmd := exec.Command("", execSpec.CompileArgs...)

		// Capture stdout
		output, err := cmd.Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(output))
	}

	cmd := exec.Command("", execSpec.RunArgs...)
	// Capture stdout
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(output))
}
