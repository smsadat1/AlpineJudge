// calculates resoruce availability using system monitor data
package scheduler

import (
	"cmp"
	"utils"
)

const OVER_SUB_FACTOR = 2
const SLOT_FLOOR = 0.0
const MEMORYLIMITMB_PER_CONTAINER = 256

func clamp[T cmp.Ordered](val, min, max T) T {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// Resource Aware Dynamic Scheduler
func RADScheduler(
	availableMemoryMB int, cpuCoreCount int, runningContainers int,
) utils.RADSDecision {

	slotBaseline := cpuCoreCount
	slotCeiling := cpuCoreCount * OVER_SUB_FACTOR

	memorySlots := float64((0.8 * float64(availableMemoryMB)) / MEMORYLIMITMB_PER_CONTAINER)
	availableSlots := clamp(min(memorySlots, float64(slotCeiling)), SLOT_FLOOR, float64(slotCeiling))
	usedSlots := runningContainers
	idleSlots := availableSlots - float64(usedSlots)

	var status string

	if idleSlots >= float64(slotBaseline) {
		status = "NORMAL"
	} else if idleSlots < float64(slotBaseline) && idleSlots > SLOT_FLOOR {
		status = "DEGRADED"
	} else if idleSlots <= float64(SLOT_FLOOR) {
		status = "CRITICAL"
	}

	decisions := utils.RADSDecision{
		AvailableSlots: int(availableSlots),
		IdleSlots:      idleSlots,
		UsedSlots:      usedSlots,
		Status:         status,
	}

	return decisions
}
