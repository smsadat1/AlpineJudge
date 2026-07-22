package unit_test

import (
	"context"
	"local/runner/scheduler"
	"testing"
	"time"
	"utils"
)

func Test_SystemMonitor(t *testing.T) {

	scheduler.GetCPUSampleFn = func() (utils.CPUSample, error) { return utils.CPUSample{}, nil }
	scheduler.CalcCPUUsageFn = func(p, c utils.CPUSample) float64 { return 25.5 }
	scheduler.GetMemoryUsageFn = func() (uint64, uint64, error) { return 16000, 4000, nil }
	scheduler.RuntimeNumCPUFn = func() int { return 8 }

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sysQueue := make(chan utils.SystemMetrics, 10)

	go func() {
		err := scheduler.SystemMonitor(ctx, 10*time.Millisecond, sysQueue)
		if err != nil {
			t.Errorf("System monitor exited with error %v\n", err)
		}
	}()

	select {
	case metric := <-sysQueue:
		if metric.CPUCoreCount != 8 {
			t.Errorf("Expected 8 CPU cores, got %d", metric.CPUCoreCount)
		}
		if metric.CPUUsagePercent != 25.5 {
			t.Errorf("Expected 25.5%% CPU usage, got %f", metric.CPUUsagePercent)
		}
		if metric.MemoryUsagePercent != 75.0 { // (16000-4000)/16000 * 100
			t.Errorf("Expected 75%% memory usage, got %f", metric.MemoryUsagePercent)
		}
		t.Logf("Successfully caught real-time channel metric payload: %+v", metric)

		// stop background loop
		cancel()

	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out waiting for SystemMonitor to send data to the Go channel")
	}
}
