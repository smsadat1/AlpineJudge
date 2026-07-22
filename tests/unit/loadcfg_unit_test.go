package tests

import (
	"testing"
	"utils"
)

func Test_LoadRunnerConfigs(t *testing.T) {
	err := utils.LoadRunnerConfigs("../artifacts/runner.config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if utils.RunCfg.RunnerID != "runner-001" ||
		utils.RunCfg.Images["c"] != "ghcr.io/smsadat1/ajgcc:v0.1.0" ||
		utils.RunCfg.Images["cc"] != "ghcr.io/smsadat1/ajgcc:v0.1.0" ||
		utils.RunCfg.Images["cpp"] != "ghcr.io/smsadat1/ajgcc:v0.1.0" ||
		utils.RunCfg.Images["py"] != "ghcr.io/smsadat1/ajpython:v0.1.0" ||
		utils.RunCfg.Images["java"] != "ghcr.io/smsadat1/ajjava:v0.1.0" ||
		utils.RunCfg.Images["go"] != "ghcr.io/smsadat1/ajgo:v0.1.0" ||
		utils.RunCfg.Images["js"] != "ghcr.io/smsadat1/ajnode:v0.1.0" {
		t.Fatal("Image configuration mismatches detected")
	}

	if utils.RunCfg.Scheduler.MemResPerc != 20 ||
		utils.RunCfg.Scheduler.OverSubFactor != 2 {
		t.Fatal("Scheduler configuration mismatch detected")
	}

	if utils.RunCfg.Limits.CPUQuota != 2 ||
		utils.RunCfg.Limits.PIDLimit != 128 ||
		utils.RunCfg.Limits.MemoryLimitMB != 1024 ||
		utils.RunCfg.Limits.NoNewPrivs != true ||
		utils.RunCfg.Limits.RORootFS != true ||
		utils.RunCfg.Limits.TimeoutSec != 300 {
		t.Fatal("Resource limit configuration mismatch detected")
	}

}
