// system monitor
package scheduler

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"utils"
)

func getCPUSample() (utils.CPUSample, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return utils.CPUSample{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 9 || fields[0] != "cpu" {
			return utils.CPUSample{}, fmt.Errorf("unexpected /proc/stat format")
		}

		u, _ := strconv.ParseUint(fields[1], 10, 64)
		n, _ := strconv.ParseUint(fields[2], 10, 64)
		s, _ := strconv.ParseUint(fields[3], 10, 64)
		i, _ := strconv.ParseUint(fields[4], 10, 64)
		io, _ := strconv.ParseUint(fields[5], 10, 64)
		irq, _ := strconv.ParseUint(fields[6], 10, 64)
		sirq, _ := strconv.ParseUint(fields[7], 10, 64)
		stl, _ := strconv.ParseUint(fields[8], 10, 64)

		return utils.CPUSample{
			User: u, Nice: n, System: s, Idle: i, Iowait: io, Irq: irq, Softirq: sirq, Steal: stl}, nil
	}

	return utils.CPUSample{}, fmt.Errorf("Empty /proc/stat")
}

func calcCPUUsage(prev, curr utils.CPUSample) float64 {
	prevIdle := prev.Idle + prev.Iowait
	currIdle := curr.Idle + curr.Iowait

	prevNonIdle := prev.User + prev.Nice + prev.System + prev.Irq + prev.Softirq + prev.Steal
	currNonIdle := curr.User + curr.Nice + curr.System + curr.Irq + curr.Softirq + curr.Steal

	prevTotal := prevIdle + prevNonIdle
	currTotal := currIdle + currNonIdle

	totalDelta := currTotal - prevTotal
	idleDelta := currIdle - prevIdle

	if totalDelta == 0 {
		return 0.0
	}
	return (float64(totalDelta-idleDelta) / float64(totalDelta)) * 100
}

// parses memory info from /proc/meminfo and returns MB
func getMemoryUsage() (total uint64, available uint64, err error) {

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	found := 0

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		if strings.HasPrefix(fields[0], "MemTotal") {
			total, _ = strconv.ParseUint(fields[1], 10, 64)
			found++
		} else if strings.HasPrefix(fields[0], "MemAvailable") {
			available, _ = strconv.ParseUint(fields[1], 10, 64)
			found++
		}

		if found == 2 {
			break
		}
	}

	if err = scanner.Err(); err != nil {
		log.Printf("Error encountered during scanning: %v", err)
		return 0, 0.0, err
	}

	// KB to MB
	return total / 1024, available / 1024, nil
}

var (
	GetCPUSampleFn   = getCPUSample
	CalcCPUUsageFn   = calcCPUUsage
	GetMemoryUsageFn = getMemoryUsage
	RuntimeNumCPUFn  = runtime.NumCPU
)

func SystemMonitor(
	ctx context.Context, interval time.Duration, sysMetricsQueue chan<- utils.SystemMetrics,
) error {

	cpuCount := RuntimeNumCPUFn()
	tick := time.NewTicker(time.Duration(interval))
	defer tick.Stop()

	log.Printf("Monitor service started with interval of: %v\n", interval)

	// initial baseline reading for CPU calculation
	prevCPU, err := GetCPUSampleFn()
	if err != nil {
		log.Printf("Error getting initial CPU state: %v\n", err)
		return err
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Exiting system monitor loop...")
			return nil
		case <-tick.C:
			currentCPU, err := GetCPUSampleFn()
			var cpuPercent float64
			if err == nil {
				cpuPercent = CalcCPUUsageFn(prevCPU, currentCPU)
				prevCPU = currentCPU
			} else {
				log.Printf("Error reading CPU: %v\n", err)
				continue
			}

			memTotal, memavail, err := GetMemoryUsageFn()
			var memUsed uint64
			var memPercent float64
			if err == nil {
				memUsed = memTotal - memavail
				memPercent = (float64(memUsed) / float64(memTotal)) * 100
			} else {
				log.Printf("Error reading memory %v\n", err)
				continue
			}

			sysMetrics := utils.SystemMetrics{
				CPUUsagePercent:    cpuPercent,
				MemoryUsagePercent: memPercent,
				AvailableMemoryMB:  int(memavail),
				CPUCoreCount:       cpuCount,
			}

			select {
			case sysMetricsQueue <- sysMetrics:
			case <-ctx.Done():
				return nil
			}
		}
	}
}
