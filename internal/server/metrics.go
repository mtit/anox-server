package server

import (
	"log"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func getSystemInfo() (int, int64) {
	cores, err := cpu.Counts(true)
	if err != nil {
		cores = runtime.NumCPU()
	}

	var memTotal int64
	if vm, err := mem.VirtualMemory(); err == nil {
		memTotal = int64(vm.Total / 1024 / 1024)
	} else {
		log.Printf("[Anox Server] Failed to read memory info: %v", err)
	}

	return cores, memTotal
}

func getSystemMetrics() (cpuPercent float64, memUsed, memTotal int64) {
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		cpuPercent = percents[0]
	} else if err != nil {
		log.Printf("[Anox Server] Failed to read CPU metrics: %v", err)
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		memTotal = int64(vm.Total / 1024 / 1024)
		memUsed = int64((vm.Total - vm.Available) / 1024 / 1024)
	} else if err != nil {
		log.Printf("[Anox Server] Failed to read memory metrics: %v", err)
	}

	return cpuPercent, memUsed, memTotal
}
