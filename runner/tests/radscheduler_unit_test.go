package tests

import (
	"local/runner/scheduler"
	"local/runner/utils"
	"testing"
)

func Test_RADScheduler(t *testing.T) {

	var rd utils.RADSDecision

	// 4GB memory, 4 CPU cores, 8 running containers

	rd = scheduler.RADScheduler(4096, 4, 4)
	if rd.Status != "NORMAL" {
		t.Logf("Expected status: %v\n", rd.Status)
	}

	rd = scheduler.RADScheduler(4096, 4, 6)
	if rd.Status != "DEGRADED" {
		t.Logf("Expected status: %v\n", rd.Status)
	}

	rd = scheduler.RADScheduler(4096, 4, 8)
	if rd.Status != "CRITICAL" {
		t.Logf("Expected status: %v\n", rd.Status)
	}
}
